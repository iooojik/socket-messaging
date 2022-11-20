package sockets

import "net"

type Server struct {
	host    string
	port    string
	netType string
	pool    map[*client]bool
}

type client struct {
	connection net.Conn
}
