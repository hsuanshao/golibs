package ianam

// TZDBResponse describe timezonedb response body
type TZDBResponse struct {
	Zones []*ZoneInfo `json:"zones"`
}

// ZoneInfo describe a timezone information
// NOTE: country code is not always unique
type ZoneInfo struct {
	Code             string       `json:"countryCode"`
	Name             string       `json:"countryName"`
	Timezone         IANATimezone `json:"zoneName"`
	GMTOffsetSeconds int          `json:"gmtOffset"`
}

type IANATimezone string

func (tz *IANATimezone) String() string {
	return string(*tz)
}

type Timezones []IANATimezone

func (t Timezones) Len() int {
	return len(t)
}

func (t Timezones) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t Timezones) Less(i, j int) bool {
	return t[i] < t[j]
}
