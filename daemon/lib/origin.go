package lib

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jbrodriguez/controlrd/daemon/common"
	"github.com/jbrodriguez/controlrd/daemon/dto"

	"github.com/vaughan0/go-ini"
)

func GetOrigin() (*dto.Origin, error) {
	var origin dto.Origin

	address, err := getIPAddress()
	if err != nil {
		origin.ErrorCode = "ipaddress"
		origin.ErrorText = err.Error()
		return &origin, err
	}

	base, ssl, err := getOriginFromVarIni(address)
	if err != nil {
		origin.ErrorCode = "varini"
		origin.ErrorText = err.Error()
		return &origin, err
	}

	origin.Host = base.Host
	origin.Protocol = base.Protocol
	origin.Port = base.Port
	origin.Name = base.Name
	origin.Address = base.Address

	if ssl {
		myservers, err := getMyUnraidNetURL()
		if err != nil {
			origin.ErrorCode = "myservers"
			origin.ErrorText = err.Error()
			return nil, err
		}
		origin.Host = myservers.Host
		origin.Port = myservers.Port
	} else {
		host, err := getNetworkNameFromHosts()
		if err != nil {
			origin.ErrorCode = "etchosts"
			origin.ErrorText = err.Error()
			return nil, err
		}

		if host != "" {
			origin.Host = host
		}
	}

	return &origin, nil
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

func getMyUnraidNetURL() (*dto.Origin, error) {
	myservers, err := ini.LoadFile(common.Myservers)
	if err != nil {
		return nil, err
	}

	remotes, _ := myservers.Get("remote", "allowedOrigins")
	origins := strings.Split(remotes, ",")
	origin := ""

	// Regular expression to match the specific URL format
	re := regexp.MustCompile(`https://\d+-\d+-\d+-\d+\.[a-f0-9]{40}\.myunraid\.net`)

	// Find the specific URL we're looking for
	for _, url := range origins {
		if re.MatchString(url) {
			origin = url
		}
	}

	host := origin
	if host == "" {
		return nil, fmt.Errorf("specific myunraid.net URL not found")
	}

	return &dto.Origin{
		Protocol: "https",
		Host:     host,
		Port:     "443",
	}, nil
}

func getOriginFromVarIni(address string) (*dto.Origin, bool, error) {
	ident, err := ini.LoadFile(common.Variables)
	if err != nil {
		return nil, false, err
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
	}, usessl != "no", nil
}

func getNetworkNameFromHosts() (string, error) {
	file, err := os.Open("/etc/hosts")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "127.0.0.1" {
			for _, field := range fields[1:] {
				if strings.Contains(field, ".local") {
					return field, nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil // Return empty string if no matching entry found
}
