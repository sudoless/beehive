package beehive_query

import (
	"strings"
	"sync"

	"go.sdls.io/beehive/pkg/beehive"
)

// Parser is used to build a beehive.HandlerFunc that will populate the context.Context with the Values.
// The query string is parsed using simple rules, and only the keys defined in fields arg.
// Parser also uses a sync.Pool to avoid allocating new Values for each pass.
func Parser(fields []string) beehive.HandlerFunc {
	m := make(map[string]int)
	for idx, f := range fields {
		m[f] = idx + 1
	}

	pool := &sync.Pool{
		New: func() any {
			return &Values{
				dict:   m,
				values: make([]string, len(fields)+1),
			}
		},
	}

	return func(ctx *beehive.Context) beehive.Responder {
		r := ctx.Request

		query := pool.Get().(*Values)

		query.parse(r.URL.RawQuery)

		ctx.WithValue(contextValuesKey{}, query)

		ctx.After(func() {
			query.reset()
			pool.Put(query)
		})

		return nil
	}
}

func (v *Values) parse(raw string) {
	var key, value string
	var idx int

	for {
		idx = strings.IndexRune(raw, '=')
		if idx == -1 {
			break
		}

		key = raw[:idx]
		raw = raw[idx+1:]

		idx = strings.IndexRune(raw, '&')
		if idx == -1 {
			value = raw
			raw = ""
		} else {
			value = raw[:idx]
			raw = raw[idx+1:]
		}

		if lookup := v.dict[key]; lookup != 0 {
			v.values[lookup] = value
		}
	}
}
