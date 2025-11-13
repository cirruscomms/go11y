package go11y

import (
	"context"
	"net/http/httputil"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ReverseProxy is a wrapper around httputil.ReverseProxy that provides methods
// to add OpenTelemetry tracing, propagation, logging, and database storage functionality.
type ReverseProxy struct {
	*httputil.ReverseProxy
}

// AddTracing wraps a httputil.ReverseProxy's transporter with OpenTelemetry instrumentation
// This allows us to capture request and response details in our telemetry data
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func (r *ReverseProxy) AddTracing() (fault error) {
	r.Transport = otelhttp.NewTransport(r.Transport)
	return nil
}

// AddPropagation wraps a httputil.ReverseProxy's transporter with OpenTelemetry propagation
// This allows us to propagate tracing context across service boundaries
// Note: Ensure that the OpenTelemetry SDK and propagation package are properly initialized before using this client
func (r *ReverseProxy) AddPropagation() (fault error) {
	r.Transport = propagateRoundTripper(r.Transport)
	return nil
}

// AddLogging wraps a httputil.ReverseProxy's transporter with logging functionality
// This allows us to log request and response details for debugging and monitoring purposes
// Note: Ensure that the logging system is properly initialized before using this client
func (r *ReverseProxy) AddLogging() (fault error) {
	r.Transport = logRoundTripper(r.Transport)
	return nil
}

// AddDBStore wraps a httputil.ReverseProxy's transporter with database storage functionality
// This allows us to store request and response details in a database for auditing and analysis purposes
// Note: Ensure that the database connection and storage system are properly initialized before using this client
func (r *ReverseProxy) AddDBStore(ctxWithObserver context.Context) (fault error) {
	r.Transport = dbStoreRoundTripper(ctxWithObserver, r.Transport)
	return nil
}
