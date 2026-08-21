package go11y_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/cirruscomms/go11y"
)

const (
	Address = "0.0.0.0"
	Port    = 9011
)

// TestRequestLoggerMiddlewareMux is intended to show that the reset that happens in the RequestLoggerMiddlewareMux does
// not cause any problems with the go11y observer in the wider context, and that the logs and stable arguments remain as
// expected.
func TestRequestLoggerMiddlewareMux(t *testing.T) {
	expectedLogs := []string{
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware_test.go", "function":"github.com/cirruscomms/go11y_test.TestRequestLoggerMiddlewareMux", "line":1}, "level": "INFO", "msg": "Server started successfully", "address": "0.0.0.0", "port": 9011 }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/go11y.go", "function":"github.com/cirruscomms/go11y.Reset", "line":1}, "level": "DEBUG", "msg": "Observer reset" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware.go", "function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1", "line":1}, "level": "DEBUG", "msg": "request received", "origin": {"client_ip": "0.0.0.0:0000", "method": "GET", "path": "/", "user_agent": "Go-http-client/1.1"}, "request_body": "" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware_test.go", "function":"github.com/cirruscomms/go11y_test.prepareServer.handleRequests.func1", "line":1}, "level": "DEBUG", "msg": "handling request", "origin": {"client_ip": "0.0.0.0:0000", "method": "GET", "path": "/", "user_agent": "Go-http-client/1.1"}, "request_body": "" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware.go", "function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1", "line":1}, "level": "DEBUG", "msg": "request processed", "origin": {"client_ip": "0.0.0.0:0000", "method": "GET", "path": "/", "user_agent": "Go-http-client/1.1"}, "request_body": "" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/go11y.go", "function":"github.com/cirruscomms/go11y.Reset", "line":1}, "level": "DEBUG", "msg": "Observer reset"}`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware.go", "function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1", "line":1}, "level": "DEBUG", "msg": "request received", "origin": {"client_ip": "0.0.0.0:0000", "method": "POST", "path": "/", "user_agent": "Go-http-client/1.1"}, "request_body": "" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware_test.go", "function":"github.com/cirruscomms/go11y_test.prepareServer.handleRequests.func1", "line":1}, "level": "DEBUG", "msg": "handling request", "origin": {"client_ip": "0.0.0.0:0000", "method": "POST", "path": "/", "user_agent": "Go-http-client/1.1"}, "request_body": "" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware.go", "function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1", "line":1}, "level": "DEBUG", "msg": "request processed", "origin": {"client_ip": "0.0.0.0:0000", "method": "POST", "path": "/", "user_agent": "Go-http-client/1.1"}, "request_body": "" }`,
		`{ "time": "2026-07-21T08:00:00Z", "source": {"file":"/middleware_test.go", "function":"github.com/cirruscomms/go11y_test.TestRequestLoggerMiddlewareMux", "line":1}, "level": "INFO", "msg": "Finished testing RequestLoggerMiddlewareMux", "address": "0.0.0.0", "port": 9011 }`,
		``,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	bufOut := new(bytes.Buffer)

	ctx, o := initGo11y(t, ctx, bufOut, bufOut)
	o.Info("logging with go11y, without stable args")

	ctx, o, err := go11y.Extend(ctx, "address", Address, "port", Port)
	if err != nil {
		t.Fatalf("could not extend context with go11y observer: %v", err)
	}

	srv := prepareServer(t, ctx)

	o.Info("logging with go11y, with stable args")

	o.Info("Testing RequestLoggerMiddlewareMux")
	bufOut.Reset()

	go func() {
		if err := srv.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("could not start test server: %v", err)
		}
	}()
	o.Info("Server started successfully")

	// create a client and use it to send requests to the server
	client := &http.Client{}

	resp, err := client.Get(fmt.Sprintf("http://%s:%d/", srv.Address, srv.Port))
	if err != nil {
		t.Errorf("could not send request to test server: %v", err)
	} else {
		_ = resp.Body.Close()
	}

	resp, err = client.Post(fmt.Sprintf("http://%s:%d/", srv.Address, srv.Port), "application/json", nil)
	if err != nil {
		t.Errorf("could not send request to test server: %v", err)
	} else {
		_ = resp.Body.Close()
	}

	// send a termination signal to the server to test graceful shutdown
	sigChan <- syscall.SIGTERM

	// wait for the server to shut down gracefully
	time.Sleep(5 * time.Second)
	o.Info("Finished testing RequestLoggerMiddlewareMux")

	compareLogs(t, bufOut, expectedLogs)
}

type ServerConfig struct {
	Address string
	Port    int
	Server  *http.Server
}

func prepareServer(t *testing.T, ctxWithObserver context.Context) (srvr ServerConfig) {
	rtr := mux.NewRouter()

	// 🔖 Get middleware for logging all requests
	requestLoggerMiddleware, err := go11y.RequestLoggerMiddlewareMux(ctxWithObserver)
	if err != nil {
		t.Fatalf("could not get request logger middleware: %v", err)
	}

	rtr.Use(go11y.SetRequestIDMiddleware, requestLoggerMiddleware)
	rtr.Path("/").Handler(handleRequests()).Methods(http.MethodGet, http.MethodPost)
	return ServerConfig{
		Address: Address,
		Port:    Port,
		Server: &http.Server{
			Addr:                         fmt.Sprintf("%s:%d", Address, Port),
			DisableGeneralOptionsHandler: true,
			Handler:                      rtr,
		},
	}
}

func handleRequests() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, o, err := go11y.Get(r.Context())
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		o.Debug("handling request")

		w.WriteHeader(http.StatusOK)
	})
}

func compareLogs(t *testing.T, bufOut *bytes.Buffer, expectedLogs []string) {
	str := bufOut.String()
	receivedLogs := strings.Split(str, "\n")
	if len(receivedLogs) != len(expectedLogs) {
		t.Errorf("expected %d logs, received %d logs", len(expectedLogs), len(receivedLogs))
	}

	done := &completedFields{}

	for i, receivedLog := range receivedLogs {
		if i >= len(expectedLogs) {
			t.Errorf("unexpected log: %s", receivedLog)
			continue
		}
		switch {
		case strings.TrimSpace(receivedLog) == "" && strings.TrimSpace(expectedLogs[i]) != "":
			t.Errorf("log %d is empty, expected: %s", i, expectedLogs[i])
			continue
		case strings.TrimSpace(receivedLog) != "" && strings.TrimSpace(expectedLogs[i]) == "":
			t.Errorf("log %d is not empty, expected empty log", i)
			continue
		case strings.TrimSpace(receivedLog) != "" && strings.TrimSpace(expectedLogs[i]) != "":

			received := map[string]any{}
			expected := map[string]any{}
			if err := json.Unmarshal([]byte(receivedLog), &received); err != nil {
				t.Errorf("could not unmarshal received log: %v\n%s", err, receivedLog)
				continue
			}
			if err := json.Unmarshal([]byte(expectedLogs[i]), &expected); err != nil {
				t.Errorf("could not unmarshal expected log: %v", err)
				continue
			}

			compareFields(t, done, i, "root", received, expected)
		}
	}

	bufOut.Reset()
}

type completedFields []string

func (c *completedFields) add(field string) {
	*c = append(*c, field)
}

func (c *completedFields) contains(field string) bool {
	for i := range *c {
		if (*c)[i] == field {
			return true
		}
	}
	return false
}

func compareFields(t *testing.T, done *completedFields, i int, parent string, received, expected map[string]any) {
	for k, v := range received {
		switch k {
		case "time", "client_ip", "line":
			// Ignore time and client_ip fields in comparison as the contents is not stable
			done.add(fmt.Sprintf("%s.%s", parent, k))
			continue
		case "origin", "source":
			if expected[k] == nil {
				t.Errorf("%d-%s) field %s: expected nil, received '%v'", i, parent, k, v)
			}
			compareFields(t, done, i, k, v.(map[string]any), expected[k].(map[string]any))
		case "request_id":
			_, err := uuid.Parse(fmt.Sprintf("%v", v))
			if err != nil {
				t.Errorf("%d-%s) field %s: expected a valid UUID, received '%v'", i, parent, k, v)
			}
		default:
			if expected[k] != v {
				t.Errorf("%d-%s) field %s: expected '%v', received '%v'", i, parent, k, expected[k], v)
			}
		}
		done.add(fmt.Sprintf("%s.%s", parent, k))
	}

	for k, v := range expected {
		if done.contains(fmt.Sprintf("%s.%s", parent, k)) {
			continue
		}

		t.Errorf("%d-%s) field %s: expected '%v', received nil", i, parent, k, v)
	}
}

type MiddleWareConfig struct{}

func (c *MiddleWareConfig) LogLevel() slog.Level {
	return slog.LevelDebug
}

func (c *MiddleWareConfig) OtelURL() string {
	return ""
}

func (c *MiddleWareConfig) ServiceName() string {
	return "test-service"
}

func (c *MiddleWareConfig) TrimPaths() []string {
	dir, err := os.Getwd()
	if err != nil {
		return []string{}
	}
	return []string{
		dir,
	}
}

func (c *MiddleWareConfig) TrimModules() []string {
	return nil
}

func initGo11y(t *testing.T, ctx context.Context, buffOut, buffErr *bytes.Buffer) (ctxWithObserver context.Context, observer *go11y.Observer) {
	cfg := &MiddleWareConfig{}

	ctx, o, err := go11y.Initialise(ctx, cfg, buffOut, buffErr)
	if err != nil {
		t.Fatalf("could not initialise go11y observer: %v", err)
	}

	return ctx, o
}
