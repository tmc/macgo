package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
)

// Server implements an RPC server with cookie-based authentication.
type Server struct {
	jar      *CookieJar
	listener net.Listener
}

// API is the RPC service.
type API struct {
	jar *CookieJar
}

// Echo echoes a message back.
func (a *API) Echo(msg string, reply *string) error {
	*reply = "echo: " + msg
	return nil
}

// GenerateCookie generates a new authentication cookie.
func (a *API) GenerateCookie(_ int, cookie *string) error {
	c, err := a.jar.GenerateCookie()
	if err != nil {
		return err
	}
	*cookie = c
	log.Printf("generated cookie %s", c[:8])
	return nil
}

// NewServer creates a new authenticated RPC server.
func NewServer() *Server {
	return &Server{
		jar: NewCookieJar(),
	}
}

// serve handles a single connection.
// It reads the cookie from the first line, validates it, then serves RPC.
func (s *Server) serve(conn net.Conn) {
	defer conn.Close()

	// Read cookie line: "COOKIE <cookie>"
	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		log.Printf("read cookie: %v", err)
		return
	}

	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "COOKIE ") {
		log.Printf("bad auth line: %q", line)
		fmt.Fprintf(conn, "ERROR bad auth\n")
		return
	}

	cookie := strings.TrimPrefix(line, "COOKIE ")
	if !s.jar.ConsumeCookie(cookie) {
		log.Printf("bad cookie %s", cookie[:min(8, len(cookie))])
		fmt.Fprintf(conn, "ERROR bad cookie\n")
		return
	}

	log.Printf("authenticated %s", cookie[:8])
	fmt.Fprintf(conn, "OK\n")

	// Serve RPC on this connection
	api := &API{jar: s.jar}
	srv := rpc.NewServer()
	srv.Register(api)
	srv.ServeConn(conn)
}

// ListenAndServe starts the server.
func (s *Server) ListenAndServe(addr string) error {
	var ln net.Listener
	var err error

	if strings.HasPrefix(addr, "/") || strings.HasPrefix(addr, "./") {
		// Unix socket
		os.Remove(addr)
		if err := os.MkdirAll(filepath.Dir(addr), 0700); err != nil {
			return err
		}
		ln, err = net.Listen("unix", addr)
		if err != nil {
			return err
		}
		os.Chmod(addr, 0600)
		log.Printf("listening on unix:%s", addr)
	} else {
		// TCP
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		log.Printf("listening on tcp:%s", ln.Addr())
	}

	s.listener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.serve(conn)
	}
}

// Close closes the server.
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
