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
	"github.com/jbrodriguez/controlrd/daemon/plugins/sensor"
	"github.com/jbrodriguez/controlrd/daemon/plugins/ups"
)

type Api struct {
	ctx *domain.Context

	listener net.Listener

	origin *dto.Origin
	sensor sensor.Sensor
	ups    ups.Ups

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

	a.sensor = a.createSensor()
	a.ups = a.createUps()

	go a.buildQRCode()

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
	case "get_info":
		reply := a.getInfo()
		resp, _ = json.Marshal(reply)

	case "get_logs":
		params := req.Params
		logType := params["logType"]
		reply := a.getLogs(logType)
		resp, _ = json.Marshal(reply)

	case "get_origin":
		reply := a.getOrigin()
		resp, _ = json.Marshal(reply)

	default:
		resp, _ = json.Marshal(map[string]string{"error": "Unsupported action"})
	}

	logger.Yellow(" sending %+v", string(resp))

	conn.Write(resp)
	conn.Write([]byte("\n"))
}

func (a *Api) createSensor() sensor.Sensor {
	s, err := sensor.IdentifySensor()
	if err != nil {
		logger.Yellow("error identifying sensor: %s", err)
	} else {
		switch s {
		case sensor.SYSTEM:
			logger.Blue("created system sensor ...")
			return sensor.NewSystemSensor()
		case sensor.IPMI:
			logger.Blue("created ipmi sensor ...")
			return sensor.NewIpmiSensor()
		}
	}

	logger.Blue("no sensor detected ...")

	return sensor.NewNoSensor()
}

func (a *Api) createUps() ups.Ups {
	logger.Blue("showing ups %t ...", a.ctx.Config.ShowUps)
	if a.ctx.Config.ShowUps {
		u, err := ups.IdentifyUps()
		if err != nil {
			logger.Yellow("error identifying ups: %s", err)
		} else {
			switch u {
			case ups.APC:
				logger.Blue("created apc ups ...")
				return ups.NewApc()
			case ups.NUT:
				logger.Blue("created nut ups ...")
				return ups.NewNut()
			}
		}
	}

	logger.Blue("no ups detected ...")

	return ups.NewNoUps()
}
