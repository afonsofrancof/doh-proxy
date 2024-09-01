package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/miekg/dns"
	"golang.org/x/net/http2"
)

const (
	// Enum-like constants for ProtocolType
	TCP int = iota
	UDP
)

// DoHProxy represents the DNS-over-HTTPS proxy
type DoHProxy struct {
	listenAddress string       // Address to listen for DNS queries on
	port          string       // Port to listen for DNS queries on
	upstreamURLs  []string     // Upstream DoH server URLs
	protocols     []int        // List of protocols to listen on
	client        *http.Client //Http client that makes the requests to the upstream servers
	logRequests   bool         // Flag that enables logging
}

// NewDoHProxy initializes a new DoHProxy instance
func NewDoHProxy(listenAddress, port string, upstreamURLs []string, protocols []int, logRequests bool) *DoHProxy {
	// HTTP client with support for HTTP/2
	transport := &http.Transport{}

	http2.ConfigureTransport(transport) // Enable HTTP/2 support

	// HTTP client for DoH requests
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second, // Set a timeout for requests
	}

	return &DoHProxy{
		listenAddress: listenAddress,
		port:          port,
		upstreamURLs:  upstreamURLs,
		client:        client,
		protocols:     protocols,
		logRequests:   logRequests,
	}
}

// HandleDNSRequest handles incoming DNS requests and forwards them to DoH servers
func (p *DoHProxy) HandleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	dnsQuery, err := r.Pack()
	if err != nil {
		log.Printf("Failed to pack DNS request: %v", err)
		return
	}

	var response *http.Response
	var currentUpstream string

	// Send the DNS query to each upstream DoH server until one succeeds
	for _, upstream := range p.upstreamURLs {
		req, _ := http.NewRequest("POST", upstream, bytes.NewBuffer(dnsQuery))
		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("Accept", "application/dns-message")

		response, err = p.client.Do(req)
		if err == nil && response.StatusCode == http.StatusOK {
			currentUpstream = upstream
			break
		}
	}
	if currentUpstream == "" {
		if p.logRequests {
			log.Printf("Failed to query any upstream")
		}
		return
	}
	if response == nil {
		log.Printf("All upstream DoH servers failed")
		dns.HandleFailed(w, r)
		return
	}
	defer response.Body.Close()

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Failed to read response from upstream: %v", err)
		dns.HandleFailed(w, r)
		return
	}

	// Unpack the DNS response and send it back to the client
	dnsResponse := new(dns.Msg)
	if err := dnsResponse.Unpack(body); err != nil {
		log.Printf("Failed to unpack DNS response: %v", err)
		dns.HandleFailed(w, r)
		return
	}
	if p.logRequests {
		log.Println("Successfully proxied request through ", currentUpstream)
	}
	w.WriteMsg(dnsResponse)

}

// Run starts the DNS server and listens for incoming queries based on the protocol
func (p *DoHProxy) Run() {
	for _, proto := range p.protocols {
		switch proto {
		case TCP:
			go func() {
				server := &dns.Server{Addr: net.JoinHostPort(p.listenAddress, p.port), Net: "tcp"}
				dns.HandleFunc(".", p.HandleDNSRequest)

				log.Printf("Starting DoH Proxy on %s:%s over TCP", p.listenAddress, p.port)
				if err := server.ListenAndServe(); err != nil {
					log.Fatalf("Failed to start DNS server (TCP): %v", err)
				}
			}()
		case UDP:
			go func() {
				server := &dns.Server{Addr: net.JoinHostPort(p.listenAddress, p.port), Net: "udp"}
				dns.HandleFunc(".", p.HandleDNSRequest)

				log.Printf("Starting DoH Proxy on %s:%s over UDP", p.listenAddress, p.port)
				if err := server.ListenAndServe(); err != nil {
					log.Fatalf("Failed to start DNS server (UDP): %v", err)
				}
			}()
		}
	}

	// Keep the main goroutine running indefinitely
	select {}
}

func main() {
	// Define flags using the flag library
	listenAddress := flag.String("l", "127.0.0.1", "Listen address for the DNS server")
	port := flag.String("p", "53", "Port for the DNS server")

	// Define flags for protocols
	tcpFlag := flag.Bool("tcp", false, "Listen on TCP")
	udpFlag := flag.Bool("udp", false, "Listen on UDP")

	// Define flag for logging
	logFlag := flag.Bool("log", false, "Log each request proxied through an upstream")

	var upstreamURLs []string

	// Custom flag for handling multiple upstream URLs
	flag.Func("u", `Upstream DoH server URL (can be specified multiple times)
Example:
    -u https://dns.quad9.net/dns-query -u https://1.1.1.1/dns-query
WARNING:
    If this is your system's default DNS resolver 
    and the server URL is a domain name, at least one other
    DNS server after this one must be specified as an IP address
    in order to resolve the domain name of the first one.`, func(value string) error {
		log.Printf("Added %s as an upstream DoH server\n", value)
		upstreamURLs = append(upstreamURLs, value)
		return nil
	})

	// Parse the flags
	flag.Parse()

	// Check if at least one upstream DoH URL is provided
	if len(upstreamURLs) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Determine which protocols to use
	var protocols []int
	if *tcpFlag {
		protocols = append(protocols, TCP)
	}
	if *udpFlag {
		protocols = append(protocols, UDP)
	}
	if !*tcpFlag && !*udpFlag {
		// Default to both if no specific flag is provided
		protocols = []int{TCP, UDP}
	}
	if *logFlag {
		log.Println("Logging requests")
	}
	// Initialize and run the DoH proxy
	proxy := NewDoHProxy(*listenAddress, *port, upstreamURLs, protocols, *logFlag)
	proxy.Run()
}
