package sockets

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func NewSocketServer(host, port, netType string) *Server {
	return &Server{
		host:    host,
		port:    port,
		netType: netType,
		pool:    make(map[*client]bool),
	}
}

func (s *Server) Run() {
	if connection, serverErr := net.Listen(s.netType, fmt.Sprintf("%s:%s", s.host, s.port)); serverErr != nil {
		panic(serverErr)
	} else {
		defer func(connection net.Listener) {
			err := connection.Close()
			if err != nil {
				panic(err)
			}
		}(connection)
		for {
			if conn, connectionErr := connection.Accept(); connectionErr != nil {
				panic(connectionErr)
			} else {
				connectedClient := &client{connection: conn}
				s.pool[connectedClient] = true
				log.Println(fmt.Sprintf("received new connection %s total connecitons: %d", conn.RemoteAddr(), s.countConnections()))
				go s.processConnection(connectedClient)
			}
		}
	}
}

func (s *Server) countConnections() int {
	total := 0
	for _, connected := range s.pool {
		if connected {
			total += 1
		}
	}
	return total
}

func (s *Server) processConnection(client *client) {
	defer func() {
		log.Println(fmt.Sprintf("client %s disconnected", client.connection.RemoteAddr()))
		err := client.connection.Close()
		if err != nil {
			panic(err)
		}
		s.pool[client] = false
	}()
	for {
		message, err := bufio.NewReader(client.connection).ReadString('\n')
		if err != nil {
			break
		}
		fmt.Print("Message Received:", message)
	}
}

func (s *Server) makeMessage(buffer []byte) string {
	return string(buffer)
}

func (c *client) getDataBytes(buffer []byte) (int, error) {
	return c.connection.Read(buffer)
}
