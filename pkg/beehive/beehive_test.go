package beehive_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sudoless/beehive/pkg/beehive"
)

func testBeesLogger(w io.Writer) beehive.HandlerFunc {
	logger := log.New(w, "", 0)

	return func(ctx *beehive.Context) beehive.Responder {
		r := ctx.Request
		id := ctx.Value("__id")

		logger.Printf("req_id=%d method=%s path=%s", id, r.Method, r.URL.Path)
		res := ctx.WithValue("logger", logger).Next()
		logger.Printf("req_id=%d status=%d", id, res.StatusCode(ctx))

		return res
	}
}

func testBeesAuther(users map[string]string) beehive.HandlerFunc {
	authMissing := &beehive.DefaultResponder{
		Message: []byte("missing auth"),
		Status:  http.StatusUnauthorized,
	}
	authUserNotFound := &beehive.DefaultResponder{
		Message: []byte("user not found"),
		Status:  http.StatusUnauthorized,
	}
	authUserBadPassword := &beehive.DefaultResponder{
		Message: []byte("bad password"),
		Status:  http.StatusUnauthorized,
	}

	return func(ctx *beehive.Context) beehive.Responder {
		user, password, ok := ctx.Request.BasicAuth()
		if !ok {
			return authMissing
		}

		psw, ok := users[user]
		if !ok {
			return authUserNotFound
		}

		if psw != password {
			return authUserBadPassword
		}

		ctx.WithValue("user", user)

		return nil
	}
}

func testBeesStatus(code int) beehive.Responder {
	return &beehive.DefaultResponder{
		Status: code,
	}
}

type BeesHive struct {
	Name string
	Bees int
}

func testBeesJson[T any](ctx *beehive.Context) T {
	return ctx.Value("json").(T)
}

func testBeesWithJsonBody[T any]() beehive.HandlerFunc {
	return func(ctx *beehive.Context) beehive.Responder {
		body, _ := io.ReadAll(ctx.Request.Body)
		var data T

		if err := json.Unmarshal(body, &data); err != nil {
			return &beehive.DefaultResponder{
				Message: []byte(fmt.Sprintf("bad json: %s", err)),
				Status:  http.StatusBadRequest,
			}
		}

		ctx.WithValue("json", data)

		return nil
	}
}

func testBeesHandleHives(hives map[string]*BeesHive) beehive.HandlerFunc {
	return func(ctx *beehive.Context) beehive.Responder {
		switch ctx.Request.Method {
		case "POST":
			hive := testBeesJson[BeesHive](ctx)
			hives[hive.Name] = &hive

			return &beehive.DefaultResponder{
				Status: http.StatusCreated,
			}
		case "GET":
			hive, ok := hives[ctx.Request.URL.Query().Get("name")]
			if !ok {
				return &beehive.DefaultResponder{
					Message: []byte("hive not found"),
					Status:  http.StatusNotFound,
				}
			}

			return &beehive.DefaultResponder{
				Message: []byte(fmt.Sprintf("%d", hive.Bees*3)),
				Status:  200,
			}
		}

		return nil
	}
}

func Test_Bees(t *testing.T) {
	t.Parallel()

	validUser := "barry"
	validPass := "honey"

	hives := make(map[string]*BeesHive)

	users := map[string]string{
		validUser: validPass,
	}

	logBuffer := bytes.NewBuffer(nil)
	rand.Seed(1)

	router := beehive.NewRouter()

	var idCounter int64
	router.Context = func(r *http.Request) context.Context {
		idCounter++
		return context.WithValue(context.Background(), "__id", idCounter)
	}

	api := router.Group("/api", testBeesLogger(logBuffer), testBeesAuther(users))
	{
		api.Handle("GET", "/health", func(ctx *beehive.Context) beehive.Responder {
			return testBeesStatus(http.StatusOK)
		})

		api.Handle("POST", "/hive", testBeesWithJsonBody[BeesHive](), testBeesHandleHives(hives))
		api.Handle("GET", "/hive/honey", testBeesHandleHives(hives))
	}

	// call health endpoint, with valid user, check ok response
	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/health", nil)
		r.SetBasicAuth(validUser, validPass)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
		}

		expectedLog := "req_id=1 method=GET path=/api/health\nreq_id=1 status=200\n"
		if logBuffer.String() != expectedLog {
			t.Errorf("expected log %q, got %q", expectedLog, logBuffer.String())
		}
		logBuffer.Reset()
	}

	// call health endpoint, with bad password, check unauthorized response
	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/health", nil)
		r.SetBasicAuth(validUser, "bad password")
		router.ServeHTTP(w, r)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status code %d, got %d", http.StatusUnauthorized, w.Code)
		}

		if w.Body.String() != "bad password" {
			t.Errorf("expected body %q, got %q", "bad password", w.Body.String())
		}

		expectedLog := "req_id=2 method=GET path=/api/health\nreq_id=2 status=401\n"
		if logBuffer.String() != expectedLog {
			t.Errorf("expected log %q, got %q", expectedLog, logBuffer.String())
		}
		logBuffer.Reset()
	}

	// create bee hive via API, check hive was created
	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/hive", strings.NewReader(`{"name":"hive1", "bees":10}`))
		r.SetBasicAuth(validUser, validPass)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status code %d, got %d", http.StatusCreated, w.Code)
		}

		if len(hives) != 1 {
			t.Errorf("expected 1 hive, got %d", len(hives))
		}

		if hives["hive1"].Name != "hive1" {
			t.Errorf("expected hive name %q, got %q", "hive1", hives["hive1"].Name)
		}

		if hives["hive1"].Bees != 10 {
			t.Errorf("expected hive bees %d, got %d", 10, hives["hive1"].Bees)
		}

		expectedLog := "req_id=3 method=POST path=/api/hive\nreq_id=3 status=201\n"
		if logBuffer.String() != expectedLog {
			t.Errorf("expected log %q, got %q", expectedLog, logBuffer.String())
		}
		logBuffer.Reset()
	}

	// check honey production from previously created hive
	{
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/hive/honey?name=hive1", nil)
		r.SetBasicAuth(validUser, validPass)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != "30" {
			t.Errorf("expected body %q, got %q", "30", w.Body.String())
		}

		expectedLog := "req_id=4 method=GET path=/api/hive/honey\nreq_id=4 status=200\n"
		if logBuffer.String() != expectedLog {
			t.Errorf("expected log %q, got %q", expectedLog, logBuffer.String())
		}
		logBuffer.Reset()
	}

	// delete hive and check honey production again
	{
		delete(hives, "hive1")

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/hive/honey?name=hive1", nil)
		r.SetBasicAuth(validUser, validPass)
		router.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status code %d, got %d", http.StatusNotFound, w.Code)
		}

		expectedLog := "req_id=5 method=GET path=/api/hive/honey\nreq_id=5 status=404\n"
		if logBuffer.String() != expectedLog {
			t.Errorf("expected log %q, got %q", expectedLog, logBuffer.String())
		}
		logBuffer.Reset()
	}
}
