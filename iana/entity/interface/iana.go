package ianaifc

import (
	"github.com/hsuanshao/golibs/ctx"

	tzm "github.com/hsuanshao/golibs/iana/entity/models"
)

// Repository describe iana tool interface
type Repository interface {
	GetTimezoneList(ctx ctx.CTX) (ianaTZs []tzm.IANATimezone)

	QueryLocation(ctx ctx.CTX, name string) (locationTZ []*tzm.ZoneInfo, err error)
}
