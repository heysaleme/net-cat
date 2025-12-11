package main

// Логика сервера.

import (
	"net"
	"sync"
)

type Server struct {
	mu       sync.Mutex
	clients  map[net.Conn]*Client
	history  []string
	join     chan *Client
	leave    chan *Client
	messages chan string
}

func NewServer() *Server {
	return &Server{
		clients:  make(map[net.Conn]*Client),
		history:  make([]string, 0),
		join:     make(chan *Client),
		leave:    make(chan *Client),
		messages: make(chan string),
	}
}

func (s *Server) Run() {
	for {
		select {
		case c := <-s.join:
			s.handleJoin(c)
		case c := <-s.leave:
			s.handleLeave(c)
		case msg := <-s.messages:
			s.handleMessage(msg)
		}
	}
}

func (s *Server) clientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}
