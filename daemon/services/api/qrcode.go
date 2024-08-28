package api

import (
	"encoding/json"

	"github.com/jbrodriguez/controlrd/daemon/logger"
	"github.com/yeqown/go-qrcode"
)

func (a *Api) buildQRCode() {
	origin := a.getOrigin()

	logger.Blue(" origin %+v", origin)

	o, err := json.Marshal(origin)
	if err != nil {
		logger.Yellow("unable to marshal origin: %s", err)
		return
	}

	qrc, err := qrcode.New(string(o))
	if err != nil {
		logger.Yellow("unable to create qrcode: %s", err)
		return
	}

	if err := qrc.Save("/tmp/qrcode.jpg"); err != nil {
		logger.Yellow("unable to save qrcode: %s", err)
		return
	}

	return
}
