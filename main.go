package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const maxClients = 10

const usage = "[USAGE]: ./TCPChat $port"

const welcomeBanner = "Welcome to TCP-Chat!\n" +
	"         _nnnn_\n" +
	"        dGGGGMMb\n" +
	"       @p~qp~~qMb\n" +
	"       M|@||@) M|\n" +
	"       @,----.JM|\n" +
	"      JS^\\__/  qKL\n" +
	"     dZP        qKRb\n" +
	"    dZP          qKKb\n" +
	"   fZP            SMMb\n" +
	"   HZM            MMMM\n" +
	"   FqM            MMMM\n" +
	" __| \".        |\\dS\"qML\n" +
	" |    `.       | `' \\Zq\n" +
	"_)      \\.___.,|     .'\n" +
	"\\____   )MMMMMP|   .'\n" +
	"     `-'       `--'\n" +
	"[ENTER YOUR NAME]: "


type Client struct {
	conn net.Conn
	name string
	ch   chan string
}

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

func (s *Server) handleJoin(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// добавляем клиента
	s.clients[c.conn] = c

	// отдельная горутина для отправки сообщений клиенту
	go s.clientWriter(c)

	// сначала отправляем историю
	for _, msg := range s.history {
		c.ch <- msg
	}

	// сообщение о присоединении
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

	// сохраняем в историю только обычные сообщения (с таймстампом)
	s.history = append(s.history, msg)

	for _, c := range s.clients {
		c.ch <- msg
	}
}

func (s *Server) clientWriter(c *Client) {
	for msg := range c.ch {
		_, err := fmt.Fprint(c.conn, msg)
		if err != nil {
			return
		}
	}
}

// количество текущих клиентов
func (s *Server) clientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

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

		// проверка лимита подключений
		if server.clientCount() >= maxClients {
			_, _ = conn.Write([]byte("Chat is full, try again later.\n"))
			_ = conn.Close()
			continue
		}

		go handleConnection(server, conn)
	}
}

func handleConnection(server *Server, conn net.Conn) {
	reader := bufio.NewReader(conn)

	// Приветствие и запрос имени
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

	// добавляем клиента в чат
	server.join <- client

	// читаем сообщения от клиента
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
			// не рассылаем пустые сообщения
			continue
		}

		timestamp := time.Now().Format("2006-01-02 15:04:05")
		msg := fmt.Sprintf("[%s][%s]:%s\n", timestamp, client.name, text)
		server.messages <- msg
	}

	// Клиент выходит
	server.leave <- client
}
