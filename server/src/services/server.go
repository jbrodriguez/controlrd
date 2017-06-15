package services

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	// "path/filepath"
	"regexp"
	// "strconv"
	"time"

	"jbrodriguez/controlr/plugin/server/src/dto"
	"jbrodriguez/controlr/plugin/server/src/lib"
	"jbrodriguez/controlr/plugin/server/src/net"
	// "jbrodriguez/controlr/plugin/src/server/model"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jbrodriguez/actor"
	"github.com/jbrodriguez/mlog"
	"github.com/jbrodriguez/pubsub"
	"github.com/labstack/echo"
	mw "github.com/labstack/echo/middleware"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/md5_crypt"
	"github.com/tredoe/osutil/user/crypt/sha256_crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"

	"golang.org/x/net/websocket"
)

const (
	apiVersion = "/api/v1"
	capacity   = 3
)

// Server type
type Server struct {
	bus      *pubsub.PubSub
	settings *lib.Settings

	engine *echo.Echo
	actor  *actor.Actor

	pool   map[uint64]*net.Connection
	data   map[string]string
	secret string

	proxy *httputil.ReverseProxy
}

// NewServer - constructor
func NewServer(bus *pubsub.PubSub, settings *lib.Settings, data map[string]string) *Server {
	server := &Server{
		bus:      bus,
		settings: settings,
		actor:    actor.NewActor(bus),
		pool:     make(map[uint64]*net.Connection),
		data:     data,
	}
	return server
}

// Start service
func (s *Server) Start() {
	mlog.Info("Starting service Server ...")

	cwd, _ := os.Getwd()

	locations := []string{
		"/usr/local/emhttp/plugins/controlr",
		"/usr/local/share/controlr",
		cwd,
		s.settings.WebDir,
	}

	location := lib.SearchFile("index.html", locations)
	if location == "" {
		msg := ""
		for _, loc := range locations {
			msg += fmt.Sprintf("%s, ", loc)
		}
		mlog.Fatalf("Unable to find index.html. Exiting now. (searched in %s)", msg)
	}

	mlog.Info("Serving files from %s", location)

	// create JWT secret
	h := sha256.Sum256([]byte(s.data["name"] + s.data["timezone"] + s.data["version"] + s.data["csrf_token"]))
	s.secret = base64.StdEncoding.EncodeToString(h[:])

	targetURL, _ := url.Parse(s.data["backend"])
	s.proxy = httputil.NewSingleHostReverseProxy(targetURL)

	s.engine = echo.New()

	s.engine.Use(mw.Logger())
	s.engine.Use(mw.Recover())
	// s.engine.Use(mw.StaticWithConfig(mw.StaticConfig{
	// 	// Root:  location,
	// 	HTML5: true,
	// }))

	// s.engine.Static("/", filepath.Join(location, "index.html"))
	s.engine.Static("/", filepath.Join(location, "index.html"))
	s.engine.Static("/img", filepath.Join(location, "img"))
	s.engine.Static("/app", filepath.Join(location, "app"))

	s.engine.GET("/version", s.getVersion)
	s.engine.POST("/login", s.login)

	s.engine.GET("/state/plugins/*", s.proxyHandler)
	s.engine.GET("/plugins/*", s.proxyHandler)

	r := s.engine.Group("/skt")
	r.Use(mw.JWTWithConfig(mw.JWTConfig{
		SigningKey:  []byte(s.secret),
		TokenLookup: "query:token",
	}))
	r.GET("/", s.handleWs)

	s.actor.Register("socket:broadcast", s.broadcast)
	go s.actor.React()

	port := fmt.Sprintf(":%s", s.settings.Port)
	go s.engine.Start(port)

	sport := fmt.Sprintf(":%s", s.settings.SPort)
	go s.engine.StartTLS(sport, filepath.Join(s.settings.CertDir, "cert.pem"), filepath.Join(s.settings.CertDir, "key.pem"))

	mlog.Info("Server started listening http on %s", port)
	mlog.Info("Server started listening https on %s", sport)
}

// Stop service
func (s *Server) Stop() {
	mlog.Info("stopped service Server ...")
}

func (s *Server) getVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"version": s.settings.Version})
}

func (s *Server) login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username != "root" {
		mlog.Warning("Must log in as root")
		return c.JSON(http.StatusUnauthorized, map[string]string{"token": "invalid"})
	}

	if !s.settings.Dev {
		// get the /etc/shadow entry for root
		shadowLine := ""
		lib.Shell("getent shadow root", func(line string) {
			shadowLine = line
		})

		re := regexp.MustCompile(`root:(\$(.*?)\$(.*?)\$.*?):`)

		saltString := ""
		actualHash := ""
		encType := ""
		for _, match := range re.FindAllStringSubmatch(shadowLine, -1) {
			actualHash = match[1]
			encType = match[2]
			saltString = match[3]
		}

		var crypto crypt.Crypter
		saltPrefix := ""
		// crypto := crypt.New(crypt.SHA256)
		// saltPrefix := sha256_crypt.MagicPrefix
		switch encType {
		case "1":
			crypto = crypt.New(crypt.MD5)
			saltPrefix = md5_crypt.MagicPrefix
			break
		case "5":
			crypto = crypt.New(crypt.SHA256)
			saltPrefix = sha256_crypt.MagicPrefix
			break
		case "6":
			crypto = crypt.New(crypt.SHA512)
			saltPrefix = sha512_crypt.MagicPrefix
			break
		default:
			mlog.Warning("Unknown encryption type: (%s)", encType)
			return c.JSON(http.StatusUnauthorized, map[string]string{"token": "invalid"})
		}

		saltString = fmt.Sprintf("%s%s", saltPrefix, saltString)

		shadowHash, err := crypto.Generate([]byte(password), []byte(saltString))
		if err != nil {
			mlog.Warning("Unable to create hash: %s", err)
			return c.JSON(http.StatusUnauthorized, map[string]string{"token": "invalid"})
		}

		if shadowHash != actualHash {
			mlog.Warning("shadowHash != actualHash")
			return c.JSON(http.StatusUnauthorized, map[string]string{"token": "invalid"})
		}
	}

	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	now := time.Now()

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["id"] = now.Unix()
	claims["name"] = username
	claims["admin"] = true
	claims["exp"] = now.Add(time.Minute * 60).Unix()

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"token": fmt.Sprintf("%s", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"token": t})
}

// WEBSOCKET handler
func (s *Server) handleWs(c echo.Context) (err error) {
	websocket.Handler(func(ws *websocket.Conn) {
		user := c.Get("user").(*jwt.Token)
		claims := user.Claims.(jwt.MapClaims)
		id := uint64(claims["id"].(float64))

		conn := net.NewConnection(id, ws, s.onMessage, s.onClose)
		s.pool[id] = conn
		conn.Read()

	}).ServeHTTP(c.Response(), c.Request())

	return nil
}

func (s *Server) onMessage(packet *dto.Packet) {
	s.bus.Pub(&pubsub.Message{Id: packet.ID, Payload: packet.Payload}, packet.Topic)
}

func (s *Server) onClose(c *net.Connection, err error) {
	if _, ok := s.pool[c.ID]; ok {
		delete(s.pool, c.ID)
	}
}

func (s *Server) broadcast(msg *pubsub.Message) {
	packet := msg.Payload.(*dto.Packet)
	if _, ok := s.pool[msg.Id]; ok {
		conn := s.pool[msg.Id]
		conn.Write(packet)
	}
}

// PROXY for images
func (s *Server) proxyHandler(c echo.Context) (err error) {
	s.proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}
