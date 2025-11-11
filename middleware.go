package go11y

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	oapimux "github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type requestIDKey string

const (
	RequestIDInstance requestIDKey = "requestID"
	RequestIDHeader   string       = "X-Swoop-RequestID"
)

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if requestID, ok := ctx.Value(RequestIDInstance).(string); ok {
		return requestID
	}

	return ""
}

func SetRequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a new request ID
		requestID := uuid.New().String()

		// Set the request ID in the context
		ctx := context.WithValue(r.Context(), RequestIDInstance, requestID)

		// Set the request ID in the response header
		w.Header().Set(RequestIDHeader, requestID)

		// Call the next handler with the new context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type Origin struct {
	ClientIP  string `json:"client_ip"`
	UserAgent string `json:"user_agent"`
	Method    string `json:"method"`
	Path      string `json:"path"`
}

// RequestLoggerMiddleware is a middleware that logs incoming HTTP requests and their details
// It extracts tracing information from the request headers and starts a new span for the request
// It also logs the request details using go11y, adding the go11y Observer to the request context in the process
func RequestLoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log&Trace the request
		prop := otel.GetTextMapPropagator()

		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		requestID := GetRequestID(ctx)

		ctx = Reset(ctx)

		args := []any{
			"origin",
			Origin{
				ClientIP:  r.RemoteAddr,
				UserAgent: r.UserAgent(),
				Method:    r.Method,
				Path:      r.URL.Path,
			},
			FieldRequestID, requestID,
		}

		var span trace.Span

		if og.cfg.OtelURL() != "" {
			tracer := otel.Tracer(requestID)

			// tracer
			opts := []trace.SpanStartOption{
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(argsToAttributes(args...)...),
			}
			_, span = tracer.Start(ctx, "HTTP "+r.Method+" "+r.URL.Path, opts...)

			args = append(args,
				FieldSpanID, span.SpanContext().SpanID(),
				FieldTraceID, span.SpanContext().TraceID(),
			)
		}

		ctx, o := Extend(ctx, args...)
		o.Debug("request received")

		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log the response
		o.Debug("request processed", args...)

		if og.cfg.OtelURL() != "" {
			span.End()
		}
	})
}

var (
	Requests     *prometheus.CounterVec   // the number of requests the calling service has handled
	RequestTimes *prometheus.HistogramVec // the amount of time the calling service has taken to handle requests
	RuntimeOpts  MetricsMiddlewareMuxOpts // the options used to initialise the metrics middleware
)

// MetricsMiddlewareMuxOpts are the options used to initialise the metrics middleware for a mux.Router
type MetricsMiddlewareMuxOpts struct {
	Service      string         // required - the name of the service being instrumented
	Router       *mux.Router    // required - the router for the service being instrumented. This is used to register the /internal/metrics endpoint.
	PathMaskFunc PathMask       // required - function to remove variable parts of the path for metrics aggregation. If nil, the path for metrics will not me masked
	Swagger      *openapi3.T    // optional - the swagger spec for the service being instrumented. This is used to get the endpoint names. If nil, the raw request paths are used.
	validRouter  routers.Router // the validated router created from the swagger spec
}

type PathMask func(path string) (maskedPath string)

func NoopPathMask(path string) (maskedPath string) {
	return path
}

// GetMetricsMiddlewareMux initialises a promhttp metrics route on the provided mux router with a path of
// /internal/metrics and returns a mux middleware that records request-count and request-time Prometheus metrics for
// incoming HTTP requests and publishes the values on the endpoint/route.
func GetMetricsMiddlewareMux(ctx context.Context, opts MetricsMiddlewareMuxOpts) (metricsMiddleware mux.MiddlewareFunc, fault error) {
	_, o := Get(ctx)

	Requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_requests_total", opts.Service),
		Help: fmt.Sprintf("Number of requests the %s service has handled", opts.Service),
	}, []string{"endpoint", "method", "status"})

	RequestTimes = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: fmt.Sprintf("%s_requests_times", opts.Service),
		Help: fmt.Sprintf("Time %s service takes to handle requests", opts.Service),
	}, []string{"endpoint", "method", "status"})

	// Register the metrics on Prometheus endpoint
	prometheus.MustRegister(Requests)
	prometheus.MustRegister(RequestTimes)

	opts.Router.Handle("/internal/metrics", promhttp.Handler()).Methods(http.MethodGet)

	if opts.Swagger != nil {
		vr, err := oapimux.NewRouter(opts.Swagger)
		if err != nil {
			o.Error("error creating oapi validation router: %+v", err, SeverityHigh)
			return nil, fmt.Errorf("could not create oapi validation router: %w", err)
		}

		opts.validRouter = vr
	}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t0 := time.Now()

			mrw := newMiddlewareResponseWriter(w)
			// Call the next handler
			next.ServeHTTP(mrw, r)

			path := r.URL.Path

			if opts.Swagger != nil {
				route, _, err := opts.validRouter.FindRoute(r)
				if err == nil && route != nil {
					if route.Operation != nil {
						path = route.Operation.OperationID
					} else {
						path = route.Path
					}
				}
			}

			if opts.PathMaskFunc != nil {
				path = opts.PathMaskFunc(path)
			}

			for key, val := range w.Header() {
				fmt.Println("Header:", key, val)
			}

			requestTime := time.Since(t0)
			Requests.WithLabelValues(path, r.Method, fmt.Sprintf("%d", mrw.statusCode)).Inc()
			RequestTimes.WithLabelValues(path, r.Method, fmt.Sprintf("%d", mrw.statusCode)).Observe(requestTime.Seconds())
		})
	}

	return mw, nil
}

type MiddlewareResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	headerWritten bool
}

func (mrw *MiddlewareResponseWriter) WriteHeader(code int) {
	if !mrw.headerWritten {
		mrw.statusCode = code
		mrw.ResponseWriter.WriteHeader(code)
		mrw.headerWritten = true
	}
}

func (mrw *MiddlewareResponseWriter) Write(b []byte) (int, error) {
	if !mrw.headerWritten {
		mrw.WriteHeader(http.StatusOK)
	}
	return mrw.ResponseWriter.Write(b)
}

func newMiddlewareResponseWriter(w http.ResponseWriter) *MiddlewareResponseWriter {
	return &MiddlewareResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}
