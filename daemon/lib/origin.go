package lib

import (
	"os"
	"strings"

	"github.com/jbrodriguez/controlrd/daemon/common"
	"github.com/jbrodriguez/controlrd/daemon/dto"

	"github.com/vaughan0/go-ini"
)

func GetOrigin() *dto.Origin {
	address, err := getIPAddress()
	if err != nil {
		return nil
	}

	origin, err := getMyUnraidNetURL()
	if origin != nil {
		origin.Address = address
		return origin
	}

	exists, err := Exists(common.Nginx)
	if err != nil {
		return nil
	}

	if exists {
		data, err := os.ReadFile(common.Nginx)
		if err != nil {
			return nil
		}

		origin := string(data)
		origin = strings.Replace(origin, "\n", "", -1)

		params := GetParams(`(?P<protocol>^[^:]*)://(?P<name>[^\.]*)\.(?P<tld>[^:]*):(?P<port>.*)`, origin)

		return &dto.Origin{
			Protocol: params["protocol"],
			Host:     params["name"],
			Port:     params["port"],
			Name:     params["name"],
			Address:  address,
		}
	} else {
		return getOriginFromFile(address)
	}
}

func getMyUnraidNetURL() (*dto.Origin, error) {
	myservers, err := ini.LoadFile(common.Myservers)
	if err != nil {
		return nil, err
	}

	remotes, _ := myservers.Get("remote", "allowedOrigins")
	origins := strings.Split(remotes, ",")
	origin := ""
	for _, o := range origins {
		origin = strings.TrimSpace(o)

		// Check if the origin ends with '.myunraid.net' and contains both an IP address and a hash
		if strings.HasSuffix(origin, ".myunraid.net") && strings.Contains(origin, "-") && strings.Contains(origin, ".") {
			origin = o
			break
		}
	}

	host := origin
	origin = strings.ReplaceAll(origin, "https://", "")
	origin = strings.ReplaceAll(origin, "myunraid.net", "")

	return &dto.Origin{
		Protocol: "https",
		Host:     host,
		Port:     "443",
		Name:     origin,
	}, nil
}

func getIPAddress() (string, error) {
	network, err := ini.LoadFile(common.Network)
	if err != nil {
		return "", err
	}

	var tmp string
	tmp, _ = network.Get("eth0", "IPADDR:0")
	ipaddress := strings.Replace(tmp, "\"", "", -1)

	return ipaddress, nil
}

func getOriginFromFile(address string) *dto.Origin {
	ident, err := ini.LoadFile(common.Variables)
	if err != nil {
		return nil
	}

	var usessl, portnossl, portssl, protocol, name, port string

	// if the key is missing, usessl, port and portssl are set to ""
	usessl, _ = ident.Get("", "USE_SSL")
	portnossl, _ = ident.Get("", "PORT")
	portssl, _ = ident.Get("", "PORTSSL")
	name, _ = ident.Get("", "NAME")

	// remove quotes from unRAID's ini file
	usessl = strings.Replace(usessl, "\"", "", -1)
	portnossl = strings.Replace(portnossl, "\"", "", -1)
	portssl = strings.Replace(portssl, "\"", "", -1)
	name = strings.Replace(name, "\"", "", -1)

	if usessl == "no" {
		protocol = "http"
		port = portnossl
	} else {
		protocol = "https"
		port = portssl
	}
	return &dto.Origin{
		Protocol: protocol,
		Host:     name,
		Port:     port,
		Name:     name,
		Address:  address,
	}
}
