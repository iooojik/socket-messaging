package sockets

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultCutSet = "\n\t\r "
)

var (
	selectRegexp = regexp.MustCompile(`connect ((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}:\d{1,5})`)
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
	// запускаем сокет-сервер
	host := fmt.Sprintf("%s:%s", s.host, s.port)
	log.Println(fmt.Sprintf("server running: %s", host))
	if connection, serverErr := net.Listen(s.netType, host); serverErr != nil {
		panic(serverErr)
	} else {
		defer func(connection net.Listener) {
			err := connection.Close()
			if err != nil {
				panic(err)
			}
		}(connection)
		// получаем все подключения и обрабатываем их
		for {
			if conn, connectionErr := connection.Accept(); connectionErr != nil {
				panic(connectionErr)
			} else {
				// проверяем, не заполнен ли пул подключений
				totalConnected := s.countConnections()
				if totalConnected >= s.maxConnectionPool {
					s.sendMessage("too many connections. try again later", conn)
					err := conn.Close()
					if err != nil {
						panic(err)
					}
					continue
				}
				// создаем нового клиента
				connectedClient := &client{
					connection:        conn,
					id:                conn.RemoteAddr().String(),
					incomeConnections: make(chan *processingConnection),
				}
				// указываем, что он подключен
				s.pool[connectedClient] = true
				s.sendMessage("connected\n", conn)
				log.Println(fmt.Sprintf("received new connection %s total connecitons: %d", conn.RemoteAddr(), totalConnected+1))
				// обрабатываем сообщения от клиента
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
		log.Println(fmt.Sprintf("client %s disconnected", cl.connection.RemoteAddr()))
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
	// запускаем обработку входящих подключений на клиента
	go s.processingIncomeConnections(cl)
	wg.Wait()
	close(cl.incomeConnections)
}

func (s *Server) processingIncomeConnections(initiator *client) {
	defer log.Println(fmt.Sprintf("closing connections %s", initiator.connection.RemoteAddr()))
	for destProcCon := range initiator.incomeConnections {
		for {
			if destProcCon.aimClient.processingConnection == nil {
				s.sendMessage("processing new connection\n", destProcCon.aimClient.connection)
				s.sendMessage("now you have permission to send 1 message\n", destProcCon.aimClient.connection)
				s.sendMessage("processing new connection\n", initiator.connection)
				break
			} else {
				s.sendMessage("waiting...\n", initiator.connection)
				time.Sleep(3 * time.Second)
			}
		}

		initiator.processingConnection = destProcCon
		destProcCon.aimClient.processingConnection = &processingConnection{
			aimClient: destProcCon.aimClient,
			parent:    initiator,
			messages:  destProcCon.messages,
		}
		for msg := range *destProcCon.messages {
			if msg.message == "close" {
				close(*destProcCon.messages)
				log.Println(fmt.Sprintf("client %s disconnected from host %s", destProcCon.aimClient.connection.RemoteAddr(), initiator.connection.RemoteAddr()))
				s.sendMessage(fmt.Sprintf("client %s disconnected\n", destProcCon.aimClient.connection.RemoteAddr()), destProcCon.aimClient.connection)
				s.sendMessage(fmt.Sprintf("disconnected from host %s\n", initiator.connection.RemoteAddr()), initiator.connection)
				break
			}
			// делаем что-то, когда кто-то подключается
			if msg.author == initiator && initiator.permToSend {
				s.sendMessage(msg.message+"\n", destProcCon.aimClient.connection)
			} else if msg.author.permToSend {
				s.sendMessage(msg.message+"\n", initiator.connection)
			}
			initiator.permToSend = !initiator.permToSend
			destProcCon.aimClient.permToSend = !destProcCon.aimClient.permToSend
		}
		initiator.processingConnection = nil
		destProcCon.aimClient.processingConnection = nil
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

func (s *Server) receiver(initiator *client) {
	// получение сообщений сервером
	for {
		receivedMsg, err := bufio.NewReader(initiator.connection).ReadString('\n')
		if err != nil {
			log.Println(fmt.Sprintf("bad message: %s", err.Error()))
		}
		receivedMsg = strings.Trim(receivedMsg, defaultCutSet)
		log.Println(fmt.Sprintf("message received from %s: %s\n", initiator.id, receivedMsg))
		// обработка команд
		switch {
		// получение списка возможных подключений
		case receivedMsg == "/list":
			connectionsMsg := s.getConnectionsList(initiator.id)
			if len(connectionsMsg) > 0 {
				s.sendMessage(fmt.Sprintf("%s\n", strings.Join(connectionsMsg, "\n")), initiator.connection)
			} else {
				s.sendMessage("no remote connections\n", initiator.connection)
			}
			break
		//	если сообщение подходит под регулярку selectRegexp,
		//	то пробуем добавляем в очередь клиентов на подключение к клиенту
		case selectRegexp.MatchString(receivedMsg):
			// initiator - кто подключается
			// conn - к кому подключаемся
			res := selectRegexp.FindAllStringSubmatch(receivedMsg, -1)
			remoteId := res[0][1]
			if remoteId != initiator.id {
				if destCon := s.getConnectionById(remoteId); destCon != nil {
					if destCon.processingConnection != nil {
						s.sendMessage("remote host processing another connection. please wait\n", initiator.connection)
					} else {
						destCon.permToSend = true
						s.sendMessage(fmt.Sprintf("received a new connection from %s to clients pool\n", initiator.connection.RemoteAddr()), destCon.connection)
						s.sendMessage("successfully connected\n", initiator.connection)
					}
					messagesChan := make(chan message)
					initiator.incomeConnections <- &processingConnection{
						parent:    initiator,
						aimClient: destCon,
						messages:  &messagesChan,
					}
					initiator.permToSend = false
				}
			} else {
				s.sendMessage(fmt.Sprintf("wrong remote id %s", remoteId), initiator.connection)
			}
			break
		default:
			if initiator.processingConnection != nil && initiator.permToSend {
				*initiator.processingConnection.messages <- message{
					author:  initiator,
					message: receivedMsg,
				}
				break
			}
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
	return string(buffer) + "\n"
}
