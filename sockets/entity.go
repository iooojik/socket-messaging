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
	Id         string
	connection net.Conn
}
