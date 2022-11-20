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
	selectRegexp = regexp.MustCompile(`calc ((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}:\d{1,5})`)
	//selectRegexp = regexp.MustCompile(`select ((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}:\d{1,5})\s+(\d+)`)
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
					id:                conn.RemoteAddr().String(),
					incomeConnections: make(chan *incomeConnectionData),
					number:            1,
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
		if b && c.id != author {
			clientsAddrs = append(clientsAddrs, c.id)
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
		cl.number += incomeConnection.number
		s.sendMessage(fmt.Sprintf("%d\n", cl.number), incomeConnection.connection.connection)
		s.sendMessage(fmt.Sprintf("%d\n", cl.number), cl.connection)
	}
}

func (s *Server) getConnectionById(id string) *client {
	for cl, connected := range s.pool {
		if connected && cl.id == id {
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
		fmt.Print(fmt.Sprintf("message received from %s: %s\n", cl.id, message))
		switch {
		case message == "/list":
			connectionsMsg := s.getConnectionsList(cl.id)
			if len(connectionsMsg) > 0 {
				s.sendMessage(fmt.Sprintf("%s\n", strings.Join(connectionsMsg, "\n")), cl.connection)
			} else {
				s.sendMessage("no remote connections\n", cl.connection)
			}
			break
		case selectRegexp.MatchString(message):
			res := selectRegexp.FindAllStringSubmatch(message, -1)
			remoteId := res[0][1]
			number := cl.number
			//number, e := strconv.Atoi(res[0][len(res[0])-1])
			//if e != nil {
			//	s.sendMessage("wrong data", cl.connection)
			//	continue
			//}
			if remoteId != cl.id {
				if conn := s.getConnectionById(remoteId); conn != nil {
					cl.incomeConnections <- &incomeConnectionData{
						connection: conn,
						number:     number,
					}
				}
			} else {
				s.sendMessage("wrong remote id", cl.connection)
			}
			break
		}
	}
}

func (s *Server) checkIdExists(check string) bool {
	for c, connected := range s.pool {
		if connected && c.id == check {
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
