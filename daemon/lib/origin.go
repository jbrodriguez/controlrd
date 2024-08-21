package lib

import (
	"os"
	"strings"

	"github.com/jbrodriguez/controlrd/daemon/common"
	"github.com/jbrodriguez/controlrd/daemon/dto"

	"github.com/vaughan0/go-ini"
)

func GetOrigin() *dto.Origin {
	exists, err := Exists(common.Nginx)
	if err != nil {
		return nil
	}

	address, err := getIPAddress()
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
