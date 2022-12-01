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
	id         string
	connection net.Conn
	// канал, который хранит id компьютеров, пытающихся подключиться к нему
	// и в порядке очереди отвечает каждому
	incomeConnections    chan *processingConnection
	processingConnection *processingConnection
	permToSend           bool
}

type message struct {
	author  *client
	message string
}

type processingConnection struct {
	parent    *client
	aimClient *client
	messages  *chan message
}
