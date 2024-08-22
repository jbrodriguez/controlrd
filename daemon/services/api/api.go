package api

import (
	"encoding/json"
	"log"
	"net"
	"os"

	"github.com/jbrodriguez/controlrd/daemon/common"
	"github.com/jbrodriguez/controlrd/daemon/domain"
	"github.com/jbrodriguez/controlrd/daemon/dto"
	"github.com/jbrodriguez/controlrd/daemon/logger"
)

type Api struct {
	ctx *domain.Context

	listener net.Listener
	origin   *dto.Origin

	// mailbox chan any
}

func Create(ctx *domain.Context) *Api {
	return &Api{
		ctx: ctx,
	}
}

func (a *Api) Run() error {
	// make sure there's no socket file
	os.Remove(common.Socket)

	go a.loop()

	return nil
}

func (a *Api) loop() {
	var err error
	a.listener, err = net.Listen("unix", common.Socket)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", common.Socket, err)
	}
	defer func() {
		a.listener.Close()
		os.Remove(common.Socket)
	}()

	logger.Blue("Listening on %s\n", common.Socket)

	for {
		conn, err := a.listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		go a.handleConnection(conn)
	}
}

func (a *Api) handleConnection(conn net.Conn) {
	defer conn.Close()

	var req dto.Request
	err := json.NewDecoder(conn).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		conn.Write([]byte(`{"error": "Invalid request"}` + "\n"))
		return
	}

	logger.LightGreen("received %+v ", req)

	var resp []byte
	switch req.Action {
	case "get_origin":
		reply := a.getOrigin()
		logger.Yellow(" sending %+v", reply)
		resp, _ = json.Marshal(reply)
		// resp, _ = json.Marshal(map[string]string{
		// 	"message": "Kernel version",
		// 	"kernel":  "4.19.76-linuxkit",
		// })
	default:
		resp, _ = json.Marshal(map[string]string{"error": "Unsupported action"})
	}

	conn.Write(resp)
	conn.Write([]byte("\n"))
}
