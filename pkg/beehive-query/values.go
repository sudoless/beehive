package beehive_query

import (
	"net/url"
	"strconv"
	"time"
)

// Values is a helper struct that will be populated with the values from the query string if and only if a
// Parser returned beehive.HandlerFunc is used in the chain. Otherwise, Request.Values will be nil. This is used
// as an optimization to avoid useless allocating. The beehive.HandlerFunc returned by Parser also uses a
// sync.Pool to avoid allocating a new Values for each request.
type Values struct {
	dict   map[string]int
	values []string
}

func (v *Values) reset() {
	for i := range v.values {
		v.values[i] = ""
	}
}

func (v *Values) Get(key string) string {
	return v.values[v.dict[key]]
}

func (v *Values) GetInt(key string) (int, error) {
	value := v.Get(key)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func (v *Values) GetDuration(key string) (time.Duration, error) {
	value := v.Get(key)
	if value == "" {
		return 0, nil
	}
	return time.ParseDuration(value)
}

func (v *Values) GetTime(key string) (time.Time, error) {
	value := v.Get(key)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func (v *Values) GetTimeFormat(key, format string) (time.Time, error) {
	value := v.Get(key)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(format, value)
}

func (v *Values) GetTimestampMs(key string) (time.Time, error) {
	value := v.Get(key)
	if value == "" {
		return time.Time{}, nil
	}

	valueInt, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.UnixMilli(valueInt), nil
}

func (v *Values) GetBool(key string) (bool, error) {
	value := v.Get(key)
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}

func (v *Values) GetFloat(key string) (float64, error) {
	value := v.Get(key)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

// ToUrlValues converts the Values struct to a standard url.Values struct.
func (v *Values) ToUrlValues() url.Values {
	urlValues := make(url.Values)
	for key, idx := range v.dict {
		urlValues[key] = []string{v.values[idx]}
	}
	return urlValues
}
