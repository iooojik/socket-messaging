package sockets

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
)

const (
	defaultCutSet = "\n\t\r "
)

var (
	selectRegexp = regexp.MustCompile(`select ((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}:\d{1,5})`)
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
					connection:        conn,
					Id:                conn.RemoteAddr().String(),
					incomeConnections: make(chan string),
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

func (s *Server) processConnection(cl *client) {
	defer func() {
		log.Println(fmt.Sprintf("cl %s disconnected", cl.connection.RemoteAddr()))
		err := cl.connection.Close()
		if err != nil {
			panic(err)
		}
		s.pool[cl] = false
	}()
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.receiver(cl)
	}()
	go s.processingIncomeConnections(cl)
	wg.Wait()
	close(cl.incomeConnections)
}

func (s *Server) processingIncomeConnections(cl *client) {
	defer log.Println("closing connection...")
	for incomeConnection := range cl.incomeConnections {
		if income := s.getConnectionById(incomeConnection); income != nil {
			s.sendMessage(fmt.Sprintf("hello from %s", cl.connection.RemoteAddr()), income.connection)
		}
	}
}

func (s *Server) getConnectionById(id string) *client {
	for cl, connected := range s.pool {
		if connected && cl.Id == id {
			return cl
		}
	}
	return nil
}

func (s *Server) receiver(cl *client) {
	//получение сообщений
	for {
		message, err := bufio.NewReader(cl.connection).ReadString('\n')
		if err != nil {
			break
		}
		message = strings.Trim(message, defaultCutSet)
		fmt.Print(fmt.Sprintf("Message Received from %s: %s\n", cl.Id, message))
		switch {
		case message == "/list":
			connectionsMsg := s.getConnectionsList(cl.Id)
			if len(connectionsMsg) > 0 {
				s.sendMessage(fmt.Sprintf("%s\n", strings.Join(connectionsMsg, "\n")), cl.connection)
			} else {
				s.sendMessage("no remote connections\n", cl.connection)
			}
			break
		case selectRegexp.MatchString(message):
			remoteId := selectRegexp.FindAllStringSubmatch(message, -1)[0][1]
			if remoteId != cl.Id && s.checkIdExists(remoteId) {
				cl.incomeConnections <- remoteId
			} else {
				s.sendMessage("wrong remote id", cl.connection)
			}
			break
		}
	}
}

func (s *Server) checkIdExists(check string) bool {
	for c, connected := range s.pool {
		if connected && c.Id == check {
			return true
		}
	}
	return false
}

func (s *Server) makeMessage(buffer []byte) string {
	return string(buffer)
}

func (c *client) getDataBytes(buffer []byte) (int, error) {
	return c.connection.Read(buffer)
}
