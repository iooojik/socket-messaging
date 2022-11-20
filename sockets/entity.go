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
	id                string
	connection        net.Conn
	incomeConnections chan *incomeConnectionData
	waiting           bool
	number            int
	// канал, который хранит id компьютеров, пытающихся подключиться к нему
	// и в порядке очереди отвечает каждому
}

type incomeConnectionData struct {
	connection *client
	number     int
}
