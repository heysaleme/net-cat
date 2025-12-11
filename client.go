package main

// Структура клиента + отправка сообщений.

import (
	"fmt"
	"net"
)

type Client struct {
	conn net.Conn
	name string
	ch   chan string
}

func (s *Server) clientWriter(c *Client) {
	for msg := range c.ch {
		_, err := fmt.Fprint(c.conn, msg)
		if err != nil {
			return
		}
	}
}
