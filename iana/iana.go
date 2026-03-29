package iana

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hsuanshao/golibs/ctx"
	ifc "github.com/hsuanshao/golibs/iana/entity/interface"
	tzm "github.com/hsuanshao/golibs/iana/entity/models"
	"github.com/hsuanshao/golibs/iana/tz"
)

var (
	ErrNameNotFound = errors.New("input name for query location zoneinfo could not found any data")
)

type impl struct {
	d         tzm.Timezones
	zoneSlice []*tzm.ZoneInfo
	codeMap   map[string][]*tzm.ZoneInfo
	nameMap   map[string][]*tzm.ZoneInfo
	m         sync.Mutex
}

func (im *impl) updateDB(ctx ctx.CTX) error {
	im.m.Lock()
	defer im.m.Unlock()

	tzSlice := []*tzm.ZoneInfo{}

	resp, err := http.Get("http://api.timezonedb.com/v2.1/list-time-zone?key=OEBZAOHG20NY&format=json")
	if err != nil {
		ctx.WithField("err", err).Error("get new timezonedb data failed")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		ctx.WithField("status code", resp.StatusCode).Warn("response status code is not 200")
		return err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.WithField("err", err).Error("read response body from timezonedb failed")
		return err
	}
	tzResp := tzm.TZDBResponse{}
	err = json.Unmarshal(respBody, &tzResp)
	if err != nil {
		ctx.WithField("err", err).Error("json unmarshl contry code database failed")
		return err
	}

	tzSlice = tzResp.Zones

	if len(im.codeMap) == 0 {
		im.codeMap = map[string][]*tzm.ZoneInfo{}
	}

	if len(im.nameMap) == 0 {
		im.nameMap = map[string][]*tzm.ZoneInfo{}
	}

	if len(im.zoneSlice) == 0 {
		im.zoneSlice = tzSlice
	}

	for _, zone := range tzSlice {
		if _, found := im.codeMap[zone.Code]; !found {
			im.codeMap[zone.Code] = []*tzm.ZoneInfo{}
		}

		im.codeMap[zone.Code] = append(im.codeMap[zone.Code], zone)

		if _, found := im.nameMap[zone.Name]; !found {
			im.nameMap[zone.Name] = []*tzm.ZoneInfo{}
		}

		im.nameMap[zone.Name] = append(im.nameMap[zone.Name], zone)

	}

	return nil
}

func NewRepository(ctx ctx.CTX) ifc.Repository {
	today := time.Now()
	modifiedTime := time.Unix(updateIANASliaceUnixSec, 0).Add(24 * 30 * time.Hour)

	if len(ianaTimeZones) == 0 || modifiedTime.Unix() < today.Unix() {
		tz.Prepare(ctx)
	}

	impl := &impl{
		d: getTimezones(),
	}

	err := impl.updateDB(ctx)
	if err != nil {
		ctx.WithField("err", err).Panic("get timezone db failed")
		return nil
	}

	return impl
}

func (im *impl) GetTimezoneList(ctx ctx.CTX) (ianaTZs []tzm.IANATimezone) {
	im.m.Lock()
	defer im.m.Unlock()
	return im.d
}

func (im *impl) QueryLocation(ctx ctx.CTX, name string) (locationTZ []*tzm.ZoneInfo, err error) {
	locationTZ = []*tzm.ZoneInfo{}
	if ziSlice, codeFound := im.codeMap[name]; codeFound {
		ctx.WithField("code", name).Info("found from country code records")
		return ziSlice, nil
	}

	if zones, nameFound := im.nameMap[name]; nameFound {
		ctx.WithField("name", name).Info("found from country name records")
		return zones, nil
	}

	for _, z := range im.zoneSlice {
		if strings.Contains(strings.ToLower(z.Code), strings.ToLower(name)) {
			locationTZ = append(locationTZ, z)
			continue
		}
		if strings.Contains(strings.ToLower(z.Name), strings.ToLower(name)) {
			locationTZ = append(locationTZ, z)
			continue
		}

	}

	if len(locationTZ) == 0 {
		ctx.WithField("name", name).Warn("query name is not found")
		return nil, ErrNameNotFound
	}

	return locationTZ, nil
}
