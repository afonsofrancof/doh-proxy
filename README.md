# DNS-over-HTTPS Proxy (`doh-proxy`)

A simple DNS-over-HTTPS (DoH) proxy server written in Go. This tool acts as an intermediary between a DNS client and one or more DNS-over-HTTPS servers, forwarding DNS queries over HTTP/2 and handling responses.

## Features

- Supports both TCP and UDP for DNS queries.
- Configurable to use multiple upstream DoH servers.
- Logs all proxied requests (optional).
- HTTP/2 support for faster and more secure communication.

## Getting Started

### Prerequisites

- **Go**: Make sure you have Go installed on your machine. You can download it from [golang.org](https://golang.org/dl/).

### Installation

1. Clone this repository:
    ```sh
    git clone https://git.olympuslab.net/afonso/doh-proxy
    cd doh-proxy
    ```

2. Build the Go executable:
    ```sh
    go build doh-proxy.go
    ```

### Usage

Run the `doh-proxy` with the following options:

```sh
./doh-proxy [options]
```

#### Options

- **-l**: Listen address for the DNS server (default: `127.0.0.1`).
- **-p**: Port for the DNS server (default: `53`).
- **-tcp**: Listen on TCP.
- **-udp**: Listen on UDP.
- **-log**: Enable logging for each request proxied through an upstream.
- **-u**: Specify upstream DoH server URLs (can be specified multiple times).

#### Example

To start the proxy on `localhost` at port `5353`, listening on both TCP and UDP (uses both by default), with two upstream DoH servers and logging enabled:
```sh
./doh-proxy -l 127.0.0.1 -p 5353 -tcp -udp -log -u https://dns.quad9.net/dns-query -u https://1.1.1.1/dns-query
```


### Important Note

- You need to run doh-proxy as root to use lower ports (such as 53).
- At least one upstream DoH server URL is required for the proxy to function.
- If this proxy is used as your system's default DNS resolver and the upstream server URL is a domain name, at least one other DNS server must be specified as an IP address to avoid circular dependency issues.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgements

- Uses the [miekg/dns](https://github.com/miekg/dns) package for DNS handling.
- Supports HTTP/2 with [golang.org/x/net/http2](https://pkg.go.dev/golang.org/x/net/http2).
