package api

import (
	"github.com/jbrodriguez/controlrd/daemon/dto"
	"github.com/jbrodriguez/controlrd/daemon/lib"
)

func (a *Api) getOrigin() *dto.Origin {
	if a.origin == nil {
		origin := lib.GetOrigin()
		if origin != nil {
			a.origin = origin
		}
	}
	return a.origin
}
