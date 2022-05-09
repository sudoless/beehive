package beehive_query

import (
	"reflect"
	"testing"
	"time"
)

func TestValues_reset(t *testing.T) {
	t.Parallel()

	q := &Values{
		dict: map[string]int{
			"":    0,
			"foo": 1,
			"bar": 2,
			"baz": 3,
		},
		values: []string{
			"",
			"123",
			"456",
			"789",
		},
	}

	q.reset()
	expected := []string{"", "", "", ""}

	if !reflect.DeepEqual(q.values, expected) {
		t.Errorf("expected %v, got %v", expected, q.values)
	}
}

func TestValues_Get(t *testing.T) {
	t.Parallel()

	query := &Values{
		dict: map[string]int{
			"time":         1,
			"date":         2,
			"float":        3,
			"int":          4,
			"duration":     5,
			"timestamp_ms": 6,
			"bool":         7,
			"time_format":  8,
		},
		values: []string{
			"",
			"12:34:56",
			"2006-01-02T15:04:05Z",
			"3.1415",
			"12",
			"10h20m30s",
			"1643028264684",
			"true",
			"02 Jan 06 15:04 MST",
		},
	}

	t.Run("dict", func(t *testing.T) {
		for k, v := range query.dict {
			out := query.Get(k)
			if query.values[v] != out {
				t.Errorf("%s: %s != %s", k, query.values[v], out)
			}
		}
	})
	t.Run("with parsing", func(t *testing.T) {
		if out, _ := query.GetBool("bool"); out != true {
			t.Errorf("unexpected value, %v", out)
		}
		if out, _ := query.GetTimestampMs("timestamp_ms"); out.UnixMilli() != 1643028264684 {
			t.Errorf("unexpected value, %v", out)
		}
		if out, _ := query.GetDuration("duration"); out.Seconds() != 37230 {
			t.Errorf("unexpected value, %v", out.Seconds())
		}
		if out, _ := query.GetInt("int"); out != 12 {
			t.Errorf("unexpected value, %v", out)
		}
		if out, _ := query.GetFloat("float"); out != 3.1415 {
			t.Errorf("unexpected value, %v", out)
		}

		outDate, _ := query.GetTime("date")
		y, mo, d := outDate.Date()
		h, mi, s := outDate.Clock()
		if y != 2006 || mo != 1 || d != 2 {
			t.Errorf("unexpected value, %v/%v/%v", y, mo, d)
		}
		if h != 15 || mi != 4 || s != 5 {
			t.Errorf("unexpected value, %v:%v:%v", h, mi, s)
		}

		if _, err := query.GetTimeFormat("time_format", time.RFC822); err != nil {
			t.Errorf("unexpected error, %v", err)
		}
	})
}

func TestValues_Get_error(t *testing.T) {
	t.Parallel()

	query := &Values{
		dict: map[string]int{
			"time":         1,
			"date":         2,
			"float":        3,
			"int":          4,
			"duration":     5,
			"timestamp_ms": 6,
			"bool":         7,
			"time_format":  8,
		},
		values: []string{
			"",
			"12/34/56",
			"2006/01/02",
			"3,1415",
			"foo",
			"1y5m",
			"1643028_t_264684",
			"yes",
			"Not a Date",
		},
	}

	if _, err := query.GetBool("time"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetTime("date"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetFloat("float"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetInt("int"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetDuration("duration"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetTimestampMs("timestamp_ms"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetBool("bool"); err == nil {
		t.Errorf("expected error")
	}
	if _, err := query.GetTimeFormat("time_format", time.RFC822); err == nil {
		t.Errorf("expected error")
	}
}

func TestValues_Get_empty(t *testing.T) {
	t.Parallel()

	query := &Values{
		dict: map[string]int{
			"time":         1,
			"date":         2,
			"float":        3,
			"int":          4,
			"duration":     5,
			"timestamp_ms": 6,
			"bool":         7,
			"time_format":  8,
		},
		values: []string{
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		},
	}

	if out, _ := query.GetBool("bool"); out != false {
		t.Errorf("unexpected value, %v", out)
	}
	if out, _ := query.GetTimestampMs("timestamp_ms"); !out.IsZero() {
		t.Errorf("unexpected value, %v", out)
	}
	if out, _ := query.GetDuration("duration"); out.Seconds() != 0 {
		t.Errorf("unexpected value, %v", out.Seconds())
	}
	if out, _ := query.GetInt("int"); out != 0 {
		t.Errorf("unexpected value, %v", out)
	}
	if out, _ := query.GetFloat("float"); out != 0.0 {
		t.Errorf("unexpected value, %v", out)
	}

	outDate, _ := query.GetTime("date")
	if !outDate.IsZero() {
		t.Errorf("unexpected value, %v", outDate)
	}

	outDate, _ = query.GetTimeFormat("time_format", time.RFC822)
	if !outDate.IsZero() {
		t.Errorf("unexpected value, %v", outDate)
	}
}

func TestValues_ToUrlValues(t *testing.T) {
	t.Parallel()

	v := &Values{
		dict: map[string]int{
			"":    0,
			"foo": 1,
			"bar": 2,
			"baz": 3,
		},
		values: []string{"", "a", "b", "c"},
	}

	urlValues := v.ToUrlValues()
	if urlValues == nil {
		t.Fatal("unexpected nil url values")
	}

	for k, value := range urlValues {
		if v.Get(k) != value[0] {
			t.Errorf("expected '%s', got '%s'", v.Get(k), value[0])
		}
	}
}
