package lib

import (
	"strings"

	"github.com/jbrodriguez/controlrd/daemon/common"
	"github.com/jbrodriguez/controlrd/daemon/dto"

	"github.com/vaughan0/go-ini"
)

func GetOrigin() (*dto.Origin, error) {
	var origin dto.Origin

	nginx, err := ini.LoadFile(common.Nginx)
	if err != nil {
		origin.ErrorCode = "nginx-state"
		origin.ErrorText = err.Error()
		return nil, err
	}

	vars, err := ini.LoadFile(common.Variables)
	if err != nil {
		origin.ErrorCode = "var-state"
		origin.ErrorText = err.Error()
		return nil, err
	}

	origin.Name = getValueOrDefault(vars, "NAME", "Tower")
	origin.Address = getValueOrDefault(nginx, "NGINX_LANIP", "")

	usessl := getValueOrDefault(vars, "USE_SSL", "no")
	if usessl == "no" {
		origin.Protocol = "http"
		origin.Host = getValueOrDefault(nginx, "NGINX_LANNAME", origin.Address)
		origin.Port = getValueOrDefault(vars, "PORT", "80")
	} else {
		origin.Protocol = "https"
		origin.Host = getValueOrDefault(nginx, "NGINX_LANFQDN", getValueOrDefault(nginx, "NGINX_LANMDNS", origin.Address))
		origin.Port = getValueOrDefault(vars, "PORTSSL", "443")
	}

	return &origin, nil
}

func getValueOrDefault(file ini.File, key string, def string) string {
	value, _ := file.Get("", key)
	value = strings.Replace(value, "\"", "", -1)
	if value == "" {
		value = def
	}
	return value
}
