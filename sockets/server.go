package sockets

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

const (
	defaultCutSet = "\n\t\r "
)

func NewSocketServer(host, port, netType string, maxConnectionPool int) *Server {
	return &Server{
		host:              host,
		port:              port,
		netType:           netType,
		pool:              make(map[*client]bool),
		maxConnectionPool: maxConnectionPool,
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
				totalConnected := s.countConnections()
				if totalConnected >= s.maxConnectionPool {
					s.sendMessage("too many connections. try again later", conn)
					err := conn.Close()
					if err != nil {
						panic(err)
					}
					continue
				}
				connectedClient := &client{
					connection: conn,
					Id:         conn.RemoteAddr().String(),
				}
				s.pool[connectedClient] = true
				log.Println(fmt.Sprintf("received new connection %s total connecitons: %d", conn.RemoteAddr(), totalConnected+1))
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

func (s *Server) sendMessage(message string, conn net.Conn) {
	_, err := conn.Write([]byte(message))
	if err != nil {
		panic(err)
	}
}

func (s *Server) getConnectionsList(author string) []string {
	var clientsAddrs []string
	for c, b := range s.pool {
		if b && c.Id != author {
			clientsAddrs = append(clientsAddrs, c.Id)
		}
	}
	return clientsAddrs
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
		message = strings.Trim(message, defaultCutSet)
		fmt.Print(fmt.Sprintf("Message Received from %s: %s", client.Id, message))
		switch message {
		case "/list":
			connectionsMsg := s.getConnectionsList(client.Id)
			if len(connectionsMsg) > 0 {
				s.sendMessage(fmt.Sprintf("%s\n", strings.Join(connectionsMsg, "\n")), client.connection)
			} else {
				s.sendMessage("no remote connections\n", client.connection)
			}
			break
		}

	}
}

func (s *Server) makeMessage(buffer []byte) string {
	return string(buffer)
}

func (c *client) getDataBytes(buffer []byte) (int, error) {
	return c.connection.Read(buffer)
}
