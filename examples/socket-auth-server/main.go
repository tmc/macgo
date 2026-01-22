package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)

	server := flag.Bool("server", false, "run server")
	addr := flag.String("addr", "./socket", "address (unix socket path or host:port)")
	cookie := flag.String("cookie", "", "authentication cookie (client mode)")
	flag.Parse()

	if *server {
		runServer(*addr)
	} else {
		runClient(*addr, *cookie)
	}
}

func runServer(addr string) {
	s := NewServer()

	// Generate an initial cookie for testing
	cookie, _ := s.jar.GenerateCookie()
	fmt.Printf("Initial cookie: %s\n", cookie)
	fmt.Printf("Connect with: %s -cookie=%s\n", os.Args[0], cookie)
	fmt.Println()

	if err := s.ListenAndServe(addr); err != nil {
		log.Fatal(err)
	}
}

func runClient(addr, cookie string) {
	if cookie == "" {
		log.Fatal("must specify -cookie")
	}

	network := "unix"
	if addr[0] != '/' && addr[0] != '.' {
		network = "tcp"
	}

	c, err := Dial(network, addr, cookie)
	if err != nil {
		log.Fatal(err)
	}
	

	log.Printf("connected and authenticated")

	// Try echo
	reply, err := c.Echo("hello world")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("echo reply: %s", reply)

	// Try generating a cookie
	newCookie, err := c.GenerateCookie()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("generated new cookie: %s", newCookie[:8])
}
