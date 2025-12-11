package main

// Только запуск и приём подключений.

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	args := os.Args[1:]
	port := "8989"

	if len(args) == 1 {
		port = args[0]
	} else if len(args) > 1 {
		fmt.Println(usage)
		return
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Printf("Listening on the port :%s\n", port)

	server := NewServer()
	go server.Run()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}

		if server.clientCount() >= maxClients {
			_, _ = conn.Write([]byte("Chat is full, try again later.\n"))
			_ = conn.Close()
			continue
		}

		go handleConnection(server, conn)
	}
}
