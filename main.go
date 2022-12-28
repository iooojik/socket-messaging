package main

import (
	"log"
	"os"
	"socket-messaging/sockets"
)

func main() {
	log.Println(os.Args)
	if len(os.Args) >= 3 {
		srv := sockets.NewSocketServer(os.Args[1], os.Args[2], "tcp", 10)
		srv.Run()
	} else {
		log.Println("host and port not provided. use app args to set them. example: ./main localhost 8080")
	}
}
