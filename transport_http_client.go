package go11y

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPClient is a wrapper around http.Client that provides methods
// to add OpenTelemetry tracing, propagation, logging, metrics, and database storage functionality.
type HTTPClient struct {
	*http.Client
}

// AddTracing wraps a http.Client's transporter with OpenTelemetry instrumentation
// This allows us to capture request and response details in our telemetry data
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func (c *HTTPClient) AddTracing() (fault error) {
	c.Transport = otelhttp.NewTransport(c.Transport)
	return nil
}

// AddPropagation wraps a http.Client's transporter with OpenTelemetry propagation
// This allows us to propagate tracing context across service boundaries
// Note: Ensure that the OpenTelemetry SDK and propagation package are properly initialized before using this client
func (c *HTTPClient) AddPropagation() (fault error) {
	c.Transport = propagateRoundTripper(c.Transport)
	return nil
}

// AddLogging wraps a http.Client's transporter with logging functionality
// This allows us to log request and response details for debugging and monitoring purposes
// Note: Ensure that the logging system is properly initialized before using this client
func (c *HTTPClient) AddLogging() (fault error) {
	c.Transport = logRoundTripper(c.Transport)
	return nil
}

// AddDBStore wraps a http.Client's transporter with database storage functionality
// This allows us to store request and response details in a database for auditing and analysis purposes
// Note: Ensure that the database connection and storage system are properly initialized before using this client
func (c *HTTPClient) AddDBStore(ctxWithObserver context.Context) (fault error) {
	_, o := Get(ctxWithObserver)

	if o.db == nil {
		return errors.New("go11y initialised with out a database - cannot add DBStore transport")
	}

	c.Transport = dbStoreRoundTripper(ctxWithObserver, c.Transport)

	return nil
}

// MetricsRecorder is a function type for recording metrics.
type MetricsRecorder func(statusCode int, method, path string, startTime time.Time)

// AddMetrics wraps a http.Client's transporter with metrics recording functionality
// $recorder is the function that actually records the metrics - if it is nil an error is returned
// This allows us to record metrics for request and response details for monitoring purposes
func (c *HTTPClient) AddMetrics(recorder MetricsRecorder, pathMaskFunc PathMask) (fault error) {
	if recorder == nil {
		return errors.New("recorder cannot be nil")
	}

	c.Transport = metricsRoundTripper(c.Transport, recorder, pathMaskFunc)

	return nil
}
