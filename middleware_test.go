package go11y_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
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
		`{"time":"2026-08-25T12:38:13.751796+08:00","level":"INFO","source":{"function":"github.com/cirruscomms/go11y_test.TestRequestLoggerMiddlewareMux","file":"/middleware_test.go","line":84},"msg":"Server started successfully","address":"0.0.0.0","port":9011}`,
		`{"time":"2026-08-25T12:38:13.756618+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":125},"msg":"Reset go11y observer for request logger"}`,
		`{"time":"2026-08-25T12:38:13.756909+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":177},"msg":"request received","origin":{"client_ip":"127.0.0.1:51888","user_agent":"Go-http-client/1.1","method":"POST","path":"/"},"request_id":"b8532d2f-3e99-42b7-b839-6fc3f2f3ecb4","request_body":"eyJkZWxheSI6MjAwMCwia2V5MSI6Iio0KiIsImtleTIiOiIqNCoifQ=="}`,
		`{"time":"2026-08-25T12:38:13.756924+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y_test.requestHandler.func1","file":"/middleware_test.go","line":189},"msg":"received POST request","origin":{"client_ip":"127.0.0.1:51888","user_agent":"Go-http-client/1.1","method":"POST","path":"/"},"request_id":"b8532d2f-3e99-42b7-b839-6fc3f2f3ecb4"}`,
		`{"time":"2026-08-25T12:38:13.756944+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y_test.requestHandler.func1","file":"/middleware_test.go","line":217},"msg":"sleeping for a bit","origin":{"client_ip":"127.0.0.1:51888","user_agent":"Go-http-client/1.1","method":"POST","path":"/"},"request_id":"b8532d2f-3e99-42b7-b839-6fc3f2f3ecb4"}`,
		`{"time":"2026-08-25T12:38:13.757363+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":125},"msg":"Reset go11y observer for request logger"}`,
		`{"time":"2026-08-25T12:38:13.757394+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":177},"msg":"request received","origin":{"client_ip":"127.0.0.1:51889","user_agent":"Go-http-client/1.1","method":"GET","path":"/"},"request_id":"98121712-5381-437f-80a5-477b7ca5c81f","request_body":""}`,
		`{"time":"2026-08-25T12:38:13.757442+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y_test.requestHandler.func1","file":"/middleware_test.go","line":189},"msg":"received GET request","origin":{"client_ip":"127.0.0.1:51889","user_agent":"Go-http-client/1.1","method":"GET","path":"/"},"request_id":"98121712-5381-437f-80a5-477b7ca5c81f"}`,
		`{"time":"2026-08-25T12:38:13.757445+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y_test.requestHandler.func1","file":"/middleware_test.go","line":237},"msg":"handled GET request","origin":{"client_ip":"127.0.0.1:51889","user_agent":"Go-http-client/1.1","method":"GET","path":"/"},"request_id":"98121712-5381-437f-80a5-477b7ca5c81f","ephemeral":"arg"}`,
		`{"time":"2026-08-25T12:38:13.75745+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":192},"msg":"request processed","origin":{"client_ip":"127.0.0.1:51889","user_agent":"Go-http-client/1.1","method":"GET","path":"/"},"request_id":"98121712-5381-437f-80a5-477b7ca5c81f"}`,
		`{"time":"2026-08-25T12:38:15.758105+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y_test.requestHandler.func1","file":"/middleware_test.go","line":234},"msg":"request body processed","origin":{"client_ip":"127.0.0.1:51888","user_agent":"Go-http-client/1.1","method":"POST","path":"/"},"request_id":"b8532d2f-3e99-42b7-b839-6fc3f2f3ecb4","delay":2000,"key1":"value1","key2":"value2"}`,
		`{"time":"2026-08-25T12:38:15.758146+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y_test.requestHandler.func1","file":"/middleware_test.go","line":237},"msg":"handled POST request","origin":{"client_ip":"127.0.0.1:51888","user_agent":"Go-http-client/1.1","method":"POST","path":"/"},"request_id":"b8532d2f-3e99-42b7-b839-6fc3f2f3ecb4","delay":2000,"key1":"value1","key2":"value2","ephemeral":"arg"}`,
		`{"time":"2026-08-25T12:38:15.758177+08:00","level":"DEBUG","source":{"function":"github.com/cirruscomms/go11y.RequestLoggerMiddlewareMux.func1.1","file":"/middleware.go","line":192},"msg":"request processed","origin":{"client_ip":"127.0.0.1:51888","user_agent":"Go-http-client/1.1","method":"POST","path":"/"},"request_id":"b8532d2f-3e99-42b7-b839-6fc3f2f3ecb4","delay":2000,"key1":"value1","key2":"value2"}`,
		`{"time":"2026-08-25T12:38:15.758724+08:00","level":"INFO","source":{"function":"github.com/cirruscomms/go11y_test.TestRequestLoggerMiddlewareMux","file":"/middleware_test.go","line":149},"msg":"Finished testing RequestLoggerMiddlewareMux","address":"0.0.0.0","port":9011}`,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	bufOut := new(bytes.Buffer)

	ctx, o := initGo11y(t, ctx, bufOut, bufOut)
	o.Info("logging with go11y, without stable args")

	ctx, o, err := go11y.Extend(ctx, "address", Address, "port", Port)
	if err != nil {
		t.Fatalf("could not extend context with go11y observer: %v", err)
	}

	// Create synchronization channels (buffered to prevent blocking)
	postStarted := make(chan struct{}, 1)
	postSleeping := make(chan struct{}, 1)
	getCompleted := make(chan struct{}, 1)

	srv := prepareServer(t, o, postStarted, postSleeping, getCompleted)

	o.Info("logging with go11y, with stable args")

	o.Info("Testing RequestLoggerMiddlewareMux")
	bufOut.Reset()

	go func() {
		if err := srv.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("could not start test server: %v", err)
		}
	}()

	// Give the server a moment to start listening
	time.Sleep(50 * time.Millisecond)

	o.Info("Server started successfully")

	// create a client and use it to send requests to the server
	client := &http.Client{}

	delay := 2000

	body := []byte(fmt.Sprintf(`{"delay": %d, "key1": "value1", "key2": "value2"}`, delay))

	// Channel to capture POST completion
	postDone := make(chan error, 1)

	// Start POST request in goroutine
	go func() {
		resp, err := client.Post(fmt.Sprintf("http://%s:%d/", srv.Address, srv.Port), "application/json", bytes.NewReader(body))
		if err != nil {
			postDone <- err
		} else {
			_ = resp.Body.Close()
			postDone <- nil
		}
	}()

	// Wait for POST to start and enter sleep phase
	select {
	case <-postStarted:
		// POST request has started
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for POST request to start")
	}

	select {
	case <-postSleeping:
		// POST request is now sleeping
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for POST request to enter sleep")
	}

	// Now send GET request (synchronously)
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/", srv.Address, srv.Port))
	if err != nil {
		t.Errorf("could not send request to test server: %v", err)
	} else {
		_ = resp.Body.Close()
	}

	// Signal that GET is complete
	close(getCompleted)

	// Wait for POST to complete
	select {
	case err := <-postDone:
		if err != nil {
			t.Errorf("POST request failed: %v", err)
		}
	case <-time.After(time.Duration(delay+1000) * time.Millisecond):
		t.Fatal("timeout waiting for POST request to complete")
	}

	// send a termination signal to the server to test graceful shutdown
	err = srv.Server.Shutdown(ctx)
	if err != nil {
		t.Errorf("could not shutdown test server: %v", err)
	}

	o.Info("Finished testing RequestLoggerMiddlewareMux")

	compareLogs(t, bufOut, expectedLogs)
}

type ServerConfig struct {
	Address string
	Port    int
	Server  *http.Server
}

func prepareServer(t *testing.T, observer *go11y.Observer, postStarted, postSleeping, getCompleted chan struct{}) (srvr ServerConfig) {
	rtr := mux.NewRouter()

	// 🔖 Get middleware for logging all requests
	requestLoggerMiddleware, err := go11y.RequestLoggerMiddlewareMux(observer)
	if err != nil {
		t.Fatalf("could not get request logger middleware: %v", err)
	}

	rtr.Use(go11y.SetRequestIDMiddleware, requestLoggerMiddleware)
	rtr.Path("/").Handler(requestHandler(postStarted, postSleeping, getCompleted)).Methods(http.MethodGet, http.MethodPost)
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

func requestHandler(postStarted, postSleeping, getCompleted chan struct{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, o, err := go11y.Get(r.Context())
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		o.Debug(fmt.Sprintf("received %s request", r.Method))

		// Signal POST has started
		if r.Method == http.MethodPost {
			postStarted <- struct{}{}
		}

		if r.Body != nil {
			defer func() {
				_ = r.Body.Close()
			}()

			b, err := io.ReadAll(r.Body) // read and discard the body
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if len(b) > 0 {
				payload := map[string]any{}
				err = json.Unmarshal(b, &payload)
				if err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				// First, handle the delay to ensure deterministic ordering
				if delay, ok := payload["delay"].(float64); ok {
					o.Debug("sleeping for a bit")
					// Signal that POST is entering sleep
					if r.Method == http.MethodPost {
						postSleeping <- struct{}{}
					}
					time.Sleep(time.Duration(delay) * time.Millisecond)
				}

				// Then extend context with all payload fields
				ctx := r.Context()
				for k, v := range payload {
					ctx, o, err = go11y.Extend(ctx, k, v)
					if err != nil {
						http.Error(w, "internal server error", http.StatusInternalServerError)
						return
					}
				}
				o.Debug("request body processed")
			}
		}
		o.Debug(fmt.Sprintf("handled %s request", r.Method), "ephemeral", "arg")

		w.WriteHeader(http.StatusOK)
	})
}

func compareLogs(t *testing.T, bufOut *bytes.Buffer, expectedLogs []string) {
	str := bufOut.String()
	receivedLogs := strings.Split(strings.TrimSpace(str), "\n")

	if len(receivedLogs) != len(expectedLogs) {
		t.Errorf("expected %d logs, received %d logs", len(expectedLogs), len(receivedLogs))
	}

	done := &completedFields{}

	passed := true

	for i, receivedLog := range receivedLogs {
		if i >= len(expectedLogs) {
			t.Errorf("unexpected log: %s", receivedLog)
			passed = false
			continue
		}
		switch {
		case strings.TrimSpace(receivedLog) == "" && strings.TrimSpace(expectedLogs[i]) != "":
			t.Errorf("log %d is empty, expected: %s", i, expectedLogs[i])
			passed = false

		case strings.TrimSpace(receivedLog) != "" && strings.TrimSpace(expectedLogs[i]) == "":
			t.Errorf("log %d is not empty, expected empty log", i)
			passed = false

		case strings.TrimSpace(receivedLog) != "" && strings.TrimSpace(expectedLogs[i]) != "":

			received := map[string]any{}
			expected := map[string]any{}
			if err := json.Unmarshal([]byte(receivedLog), &received); err != nil {
				t.Errorf("could not unmarshal received log: %v\n%s", err, receivedLog)
				passed = false
				continue
			}
			if err := json.Unmarshal([]byte(expectedLogs[i]), &expected); err != nil {
				t.Errorf("could not unmarshal expected log: %v", err)
				passed = false
				continue
			}

			child := compareFields(t, done, i, "root", received, expected)

			if !child {
				passed = false
			}
		}
	}

	if !passed {
		err := os.WriteFile("middleware_log.tmp", []byte(str), 0o644)
		if err != nil {
			t.Errorf("could not write received log to file: %v", err)
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

func compareFields(t *testing.T, done *completedFields, i int, parent string, received, expected map[string]any) bool {
	passed := true
	for k, v := range received {
		if expected[k] == nil {
			passed = false
			t.Errorf("%d-%s) field %s: expected nil, received '%v'", i, parent, k, v)
		}

		switch k {
		case "origin", "source": // Break down the origin and source fields into their subfields and compare them
			if expected[k] == nil {
				passed = false
				t.Errorf("%d-%s) field %s: expected nil, received '%v'", i, parent, k, v)
			}
			children := compareFields(t, done, i, k, v.(map[string]any), expected[k].(map[string]any))
			if !children {
				passed = false
			}
		case "client_ip": // Don't compare, just check that is is an IP address and port
			rex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}:\d+$`)
			if !rex.MatchString(fmt.Sprintf("%v", v)) {
				passed = false
				t.Errorf("%d-%s) field %s: expected a valid IP address and port, received '%v'", i, parent, k, v)
			}
		case "line": // Don't compare, just check that is is a number
			_, err := strconv.Atoi(fmt.Sprintf("%v", v))
			if err != nil {
				passed = false
				t.Errorf("%d-%s) field %s: expected a number, received '%v'", i, parent, k, v)
			}

		case "time": // Don't compare, just check that is is a valid time
			_, err := time.Parse(time.RFC3339, fmt.Sprintf("%v", v))
			if err != nil {
				passed = false
				t.Errorf("%d-%s) field %s: expected a valid time, received '%v'", i, parent, k, v)
			}
		case "request_id":
			_, err := uuid.Parse(fmt.Sprintf("%v", v))
			if err != nil {
				passed = false
				t.Errorf("%d-%s) field %s: expected a valid UUID, received '%v'", i, parent, k, v)
			}
		default:
			if expected[k] != v {
				passed = false
				t.Errorf("%d-%s) field %s: expected '%v', received '%v'", i, parent, k, expected[k], v)
			}
		}
		done.add(fmt.Sprintf("%s.%s", parent, k))
	}

	for k, v := range expected {
		if done.contains(fmt.Sprintf("%s.%s", parent, k)) {
			continue
		}

		passed = false
		t.Errorf("%d-%s) field %s: expected '%v', received nil", i, parent, k, v)
	}
	return passed
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
