package main

import (
	"bufio"
	"fmt"
	"net"
	"net/rpc"
	"strings"
)

// Client wraps an authenticated RPC client connection.
type Client struct {
	conn   net.Conn
	client *rpc.Client
}

// Dial connects to the server and authenticates with a cookie.
func Dial(network, addr, cookie string) (*Client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	// Send cookie
	fmt.Fprintf(conn, "COOKIE %s\n", cookie)

	// Read response
	br := bufio.NewReader(conn)
	resp, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}

	resp = strings.TrimSpace(resp)
	if resp != "OK" {
		conn.Close()
		return nil, fmt.Errorf("auth failed: %s", resp)
	}

	// Create RPC client
	client := rpc.NewClient(conn)

	return &Client{
		conn:   conn,
		client: client,
	}, nil
}

// Echo calls the Echo RPC method.
func (c *Client) Echo(msg string) (string, error) {
	var reply string
	err := c.client.Call("API.Echo", msg, &reply)
	return reply, err
}

// GenerateCookie calls the GenerateCookie RPC method.
func (c *Client) GenerateCookie() (string, error) {
	var cookie string
	err := c.client.Call("API.GenerateCookie", 0, &cookie)
	return cookie, err
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
