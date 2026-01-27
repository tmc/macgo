# Socket Auth Server

An example of cookie-based authentication for RPC servers, inspired by iTerm2's Python API authentication scheme.

## How It Works

The authentication pattern is simple:

1. **Server generates cookies**: Random 128-bit values stored in a cookie jar
2. **Client authenticates**: Sends `COOKIE <value>` as first line
3. **Server validates**: Checks cookie exists and hasn't been used
4. **RPC communication**: After auth, standard net/rpc protocol

## iTerm2's Authentication Scheme

iTerm2 uses a similar pattern for its Python API:

- **Socket**: Unix domain socket at `~/Library/Application Support/iTerm2/private/socket`
- **Protocol**: WebSocket with custom `x-iterm2-cookie` header
- **Cookies**: 128-bit random numbers, single-use
- **Security**: macOS Automation permissions required, or opt-in to disable

Our implementation simplifies this to plain RPC with a text-based auth handshake.

## Usage

Start the server:

```bash
$ go run . -server
Initial cookie: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
Connect with: ./socket-auth-server -cookie=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6

listening on unix:./socket
```

Connect with a client:

```bash
$ go run . -cookie=a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
connected and authenticated
echo reply: echo: hello world
generated new cookie: b2c3d4e5
```

## API Methods

- `Echo(msg string) string` - Echo a message back
- `GenerateCookie() string` - Generate a new authentication cookie

## Implementation Notes

The server follows these principles:

- **Simple protocol**: Text-based auth, then standard RPC
- **Single-use cookies**: Each cookie can only authenticate once
- **Concurrent-safe**: Cookie jar uses sync.RWMutex
- **Clean code**: Following Russ Cox style - simple, clear, effective

## Comparison to iTerm2

| Feature | iTerm2 | This Example |
|---------|--------|--------------|
| Transport | WebSocket | net/rpc |
| Auth | HTTP header | Text line |
| Socket | Unix only | Unix or TCP |
| Protocol | Protobuf | gob |
| Cookie size | 128-bit | 128-bit |
| Cookie reuse | Single-use | Single-use |

## Files

- `cookiejar.go` - Cookie generation and validation
- `server.go` - RPC server with authentication
- `client.go` - RPC client with authentication
- `main.go` - Server and client demo
