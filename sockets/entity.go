package sockets

import "net"

type Server struct {
	host              string
	port              string
	netType           string
	pool              map[*client]bool
	maxConnectionPool int
}

type client struct {
	Id                string
	connection        net.Conn
	incomeConnections chan string
	// канал, который хранит id компьютеров, пытающихся подключиться к нему
	// и в порядке очереди отвечает каждому
}
