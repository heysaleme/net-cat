package main

// Join, Leave, Message и handleConnection.

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

func (s *Server) handleJoin(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clients[c.conn] = c
	go s.clientWriter(c)

	for _, msg := range s.history {
		c.ch <- msg
	}

	joinMsg := fmt.Sprintf("%s has joined our chat...\n", c.name)
	for _, cl := range s.clients {
		if cl != c {
			cl.ch <- joinMsg
		}
	}
}

func (s *Server) handleLeave(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[c.conn]; !ok {
		return
	}

	delete(s.clients, c.conn)
	close(c.ch)
	_ = c.conn.Close()

	leaveMsg := fmt.Sprintf("%s has left our chat...\n", c.name)
	for _, cl := range s.clients {
		cl.ch <- leaveMsg
	}
}

func (s *Server) handleMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = append(s.history, msg)

	for _, c := range s.clients {
		c.ch <- msg
	}
}

func handleConnection(server *Server, conn net.Conn) {
	reader := bufio.NewReader(conn)

	_, err := fmt.Fprint(conn, welcomeBanner)
	if err != nil {
		_ = conn.Close()
		return
	}

	nameLine, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return
	}

	name := strings.TrimSpace(nameLine)
	if name == "" {
		_, _ = conn.Write([]byte("Name cannot be empty. Bye.\n"))
		_ = conn.Close()
		return
	}

	client := &Client{
		conn: conn,
		name: name,
		ch:   make(chan string),
	}

	server.join <- client

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println("read error:", err)
			}
			break
		}

		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		msg := fmt.Sprintf("[%s][%s]:%s\n", timestamp, client.name, text)
		server.messages <- msg
	}

	server.leave <- client
}
