package services

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jbrodriguez/actor"
	"github.com/jbrodriguez/mlog"
	"github.com/jbrodriguez/pubsub"
	ini "github.com/vaughan0/go-ini"

	"controlr/plugin/server/src/dto"
	"controlr/plugin/server/src/lib"
	"controlr/plugin/server/src/model"
	"controlr/plugin/server/src/plugins/sensor"
	"controlr/plugin/server/src/plugins/ups"
	"controlr/plugin/server/src/specific"
)

var iniPrefs = "/boot/config/plugins/dynamix/dynamix.cfg"
var sleepBin = "/usr/local/emhttp/plugins/dynamix.s3.sleep/include/SleepMode.php"

// Core service
type Core struct {
	bus      *pubsub.PubSub
	settings *lib.Settings

	actor *actor.Actor

	client *http.Client
	state  *model.State

	manager     specific.Manager
	logLocation map[string]string

	info    dto.Info
	watcher *fsnotify.Watcher

	ups    ups.Ups
	sensor sensor.Sensor
}

// NewCore - constructor
func NewCore(bus *pubsub.PubSub, settings *lib.Settings, state *model.State) *Core {
	core := &Core{
		bus:      bus,
		settings: settings,
		actor:    actor.NewActor(bus),
		state:    state,
		manager:  specific.NewManager(state.Version),
		logLocation: map[string]string{
			"system": "/var/log/syslog",
			"docker": "/var/log/docker.log",
			"vm":     "/var/log/libvirt/libvirtd.log",
		},
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	core.client = &http.Client{Timeout: time.Second * 10, Transport: tr}

	return core
}

// Start service
func (c *Core) Start() error {
	mlog.Info("starting service Core ...")

	c.actor.Register("model/REFRESH", c.refresh)
	c.actor.Register("model/UPDATE_USER", c.updateUser)
	c.actor.Register("api/GET_LOG", c.getLog)
	c.actor.Register("api/GET_INFO", c.getInfo)
	c.actor.Register("api/GET_MAC", c.getMac)
	c.actor.Register("api/GET_PREFS", c.getPrefs)

	wake := _getMac()
	prefs := _getPrefs()
	features := _getFeatures()

	c.sensor = c.createSensor()
	c.ups = c.createUps()
	samples := append(c.sensor.GetReadings(prefs), c.ups.GetStatus()...)

	c.info = dto.Info{
		Version:  2,
		Wake:     wake,
		Prefs:    prefs,
		Samples:  samples,
		Features: features,
	}

	var err error
	c.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		mlog.Fatal(err)
	}

	go func() {
		for {
			select {
			case event := <-c.watcher.Events:
				mlog.Info("event: %s", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					mlog.Info("modified file: %s", event.Name)
					c.info.Prefs = _getPrefs()
				}
			case err3 := <-c.watcher.Errors:
				mlog.Warning("Error:", err3)
			}
		}
	}()

	err = c.watcher.Add(iniPrefs)
	if err != nil {
		mlog.Fatal(err)
	}

	go c.actor.React()

	return nil
}

// Stop service
func (c *Core) Stop() {
	if c.watcher != nil {
		if err := c.watcher.Close(); err != nil {
			mlog.Warning("error closing watcher: %s", err)
		}
	}

	mlog.Info("stopped service Core ...")
}

// PLUGIN APP HANDLERS
func (c *Core) refresh(msg *pubsub.Message) {
	go func() {
		_dockers, err := lib.Get(c.client, c.state.Host, "/Docker")
		if err != nil {
			mlog.Warning("Unable to get dockers: %s", err)
			outbound := &dto.Packet{Topic: "model/ACCESS_ERROR", Payload: fmt.Sprintf("Unable to get unRAID state (dockers): %s", err)}
			c.bus.Pub(&pubsub.Message{Id: msg.Id, Payload: outbound}, "socket:broadcast")
			return
		}

		_vms, err := lib.Get(c.client, c.state.Host, "/VMs")
		if err != nil {
			mlog.Warning("Unable to get vms: %s", err)
			outbound := &dto.Packet{Topic: "model/ACCESS_ERROR", Payload: fmt.Sprintf("Unable to get unRAID state (vms): %s", err)}
			c.bus.Pub(&pubsub.Message{Id: msg.Id, Payload: outbound}, "socket:broadcast")
			return
		}

		_users, err := lib.Get(c.client, c.state.Host, "/state/users.ini")
		if err != nil {
			mlog.Warning("Unable to get users.ini: %s", err)
			outbound := &dto.Packet{Topic: "model/ACCESS_ERROR", Payload: fmt.Sprintf("Unable to get unRAID state (users): %s", err)}
			c.bus.Pub(&pubsub.Message{Payload: outbound}, "socket:broadcast")
			return
		}

		mlog.Info("Getting users ...")
		users := c.manager.GetUsers(_users)
		mlog.Info("Got %d users", len(users))
		mlog.Info("Getting apps ...")
		apps := c.manager.GetApps(_dockers, _vms)
		mlog.Info("Got %d apps", len(apps))

		outbound := &dto.Packet{Topic: "model/REFRESHED", Payload: map[string]interface{}{"users": users, "apps": apps}}
		c.bus.Pub(&pubsub.Message{Id: msg.Id, Payload: outbound}, "socket:broadcast")
	}()
}

func (c *Core) updateUser(msg *pubsub.Message) {
	args := msg.Payload.(map[string]interface{})

	data := map[string]string{
		"userName":    args["name"].(string),
		"userDesc":    args["perms"].(string),
		"cmdUserEdit": "Apply",
	}
	if c.state.CsrfToken != "" {
		data["csrf_token"] = c.state.CsrfToken
	}

	_, err := lib.Post(c.client, c.state.Host, "/update.htm", data)
	if err != nil {
		mlog.Warning("Unable to post updateUser: %s", err)
		outbound := &dto.Packet{Topic: "model/ACCESS_ERROR", Payload: "Unable to update User"}
		c.bus.Pub(&pubsub.Message{Payload: outbound}, "socket:broadcast")
		return
	}

	outbound := &dto.Packet{Topic: "model/USER_UPDATED", Payload: map[string]interface{}{"status": "ok"}}
	c.bus.Pub(&pubsub.Message{Id: msg.Id, Payload: outbound}, "socket:broadcast")
}

// API HANDLERS
func (c *Core) getLog(msg *pubsub.Message) {
	logType := msg.Payload.(string)

	log := make([]string, 0)

	exists, err := lib.Exists(c.logLocation[logType])
	if err != nil {
		mlog.Warning("Unable to check for log existence: %s", err)
		msg.Reply <- log
		return
	}

	if !exists {
		mlog.Warning("Log %s is not present in the system", logType)
		msg.Reply <- log
		return
	}

	cmd := "tail -n 40 " + c.logLocation[logType]

	lib.Shell(cmd, func(line string) {
		log = append(log, line)
	})

	msg.Reply <- log
}

func (c *Core) getInfo(msg *pubsub.Message) {
	c.info.Samples = append(c.sensor.GetReadings(c.info.Prefs), c.ups.GetStatus()...)
	msg.Reply <- c.info
}

func (c *Core) getMac(msg *pubsub.Message) {
	msg.Reply <- _getMac()
}

func (c *Core) getPrefs(msg *pubsub.Message) {
	msg.Reply <- _getPrefs()
}

func (c *Core) createSensor() sensor.Sensor {
	s, err := sensor.IdentifySensor()
	if err != nil {
		mlog.Warning("Error identify system temp: %s", err)
	} else {
		switch s {
		case sensor.SYSTEM:
			return sensor.NewSystemSensor()
		case sensor.IPMI:
			return sensor.NewIpmiSensor()
		}
	}

	return sensor.NewNoSensor()
}

func (c *Core) createUps() ups.Ups {
	if c.settings.ShowUps {
		u, err := ups.IdentifyUps()
		if err != nil {
			mlog.Warning("Error identifying UPS: %s", err)
		} else {
			switch u {
			case ups.APC:
				return ups.NewApc()
			case ups.NUT:
				return ups.NewNut()
			}
		}
	}

	return ups.NewNoUps()
}

func _getMac() dto.Wake {
	wake := dto.Wake{
		Mac:       "",
		Broadcast: "255.255.255.255",
	}

	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		// mlog.Info("[%s] = %s", iface.Name, iface.HardwareAddr)
		if iface.Name == "eth0" {
			wake.Mac = iface.HardwareAddr.String()
			break
		}
	}

	return wake
}

func _getPrefs() dto.Prefs {
	prefs := dto.Prefs{
		Number: ".,",
		Unit:   "C",
	}

	file, err := ini.LoadFile(iniPrefs)
	if err != nil {
		mlog.Warning("Unable to load/parse prefs file (%s): %s", iniPrefs, err)
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

func _getFeatures() map[string]bool {
	features := make(map[string]bool)

	// is sleep available ?
	exists, err := lib.Exists(sleepBin)
	if err != nil {
		mlog.Warning("getfeatures:sleep:(%s)", err)
	}

	features["sleep"] = exists

	return features
}
