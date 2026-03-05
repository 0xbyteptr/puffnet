# PuffNet

PuffNet is a simple, secure tunneling service built on WebSockets. It allows you to expose local services to the internet through a central host, with domain ownership verified by Ed25519 signatures.

## Features

- **Secure Tunnels**: Uses Ed25519 key pairs to verify domain ownership.
- **WebSocket Based**: Works through most firewalls and proxies that support WebSockets.
- **Easy to Use**: Simple CLI for hosting, serving, and accessing services.
- **Automatic Registration**: The first client to register a domain "claims" it on the host.

## Commands

### `keygen`
Generate your cryptographic identity keys (`puff.key` and `puff.pub`).
```bash
go run . keygen
```

### `host`
Start the central coordinator server.
```bash
go run . host 8008
```
The host maintains a mapping of domains to public keys in `ownership.json`. Once a domain is claimed, only the holder of the corresponding private key can serve it.

### `serve`
Expose a local service to the PuffNet host.
```bash
go run . serve -d my-site.meow -h localhost:3000
```
- `-d`: The domain to register (e.g., `my-site.meow`).
- `-h`: The local target host and port (default: `localhost:3000`).
- `-s`: The PuffNet WebSocket server URL (default: `ws://ssh.byteptr.xyz:8008/ws`).

### `get`
Fetch a resource from a registered tunnel.
```bash
go run . get my-site.meow/index.html
```
- Supports paths in the domain string (e.g., `domain.meow/api/v1`).
- Uses the default server `wss://puff.vapma.wtf/ws` unless overridden with `-s`.

### `post`
Send a POST request to a registered tunnel.
```bash
go run . post my-site.meow/api -d '{"hello": "world"}'
```
- `-d`: The POST body payload.
- Supports paths in the domain string.

## Security

PuffNet ensures that your public domain cannot be hijacked. When you run `serve`, the client signs the domain name with your private key. The host verifies this signature against the public key stored in its `ownership.json`. If the domain is new, the host locks it to your public key.

## Development

To test locally:

1. **Start the Host**: `go run . host 8008`
2. **Start a Local Server**: (e.g., `python3 -m http.server 3000`)
3. **Generate Keys**: `go run . keygen`
4. **Start the Tunnel**: `go run . serve -d local.test -h localhost:3000 -s ws://localhost:8008/ws`
5. **Request Data**: `go run . get local.test/index.html -s ws://localhost:8008/ws`
