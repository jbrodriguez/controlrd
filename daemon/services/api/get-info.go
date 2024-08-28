package api

import (
	"net"
	"strings"

	"github.com/jbrodriguez/controlrd/daemon/common"
	"github.com/jbrodriguez/controlrd/daemon/dto"
	"github.com/jbrodriguez/controlrd/daemon/lib"
	"github.com/jbrodriguez/controlrd/daemon/logger"
	"github.com/vaughan0/go-ini"
)

func (a *Api) getInfo() *dto.Info {
	prefs := getPrefs()

	sensorReadings := a.sensor.GetReadings(prefs)
	upsReadings := a.ups.GetStatus()

	samples := append(sensorReadings, upsReadings...)

	return &dto.Info{
		Version:  2,
		Wake:     getMac(),
		Prefs:    prefs,
		Samples:  samples,
		Features: getFeatures(),
	}
}

func getMac() dto.Wake {
	wake := dto.Wake{
		Mac:       "",
		Broadcast: "255.255.255.255",
	}

	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Name == "eth0" {
			wake.Mac = iface.HardwareAddr.String()
			break
		}
	}

	return wake
}

func getPrefs() dto.Prefs {
	prefs := dto.Prefs{
		Number: ".,",
		Unit:   "C",
	}

	file, err := ini.LoadFile(common.Prefs)
	if err != nil {
		logger.Yellow("unable to load/parse prefs file (%s): %s", common.Prefs, err)
		return prefs
	}

	for key, value := range file["display"] {
		if key == "number" {
			prefs.Number = strings.Replace(value, "\"", "", -1)
		}

		if key == "unit" {
			prefs.Unit = strings.Replace(value, "\"", "", -1)
		}
	}

	return prefs
}

func getFeatures() map[string]bool {
	features := make(map[string]bool)

	// is sleep available ?
	exists, err := lib.Exists(common.Sleep)
	if err != nil {
		logger.Yellow("getfeatures:sleep:(%s)", err)
	}

	features["sleep"] = exists

	return features
}
