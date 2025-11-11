package go11y

import (
	"errors"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// AddTracingToHTTPClient wraps a http.Client's transporter with OpenTelemetry instrumentation
// If the provided $httpClient is nil, an error is returned
// This allows us to capture request and response details in our telemetry data
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func AddTracingToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	// Wrap the existing transport with OpenTelemetry tracing
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)
	return nil
}

// AddPropagationToHTTPClient wraps a http.Client's transporter with OpenTelemetry propagation
// If the provided $httpClient is nil, an error is returned
// This allows us to propagate tracing context across service boundaries
// Note: Ensure that the OpenTelemetry SDK and propagation package are properly initialized before using this client
func AddPropagationToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	// Wrap the existing transport with OpenTelemetry tracing
	httpClient.Transport = propagateRoundTripper(httpClient.Transport)
	return nil
}

// AddLoggingToHTTPClient wraps a http.Client's transporter with logging functionality
// If the provided $httpClient is nil, an error is returned
// This allows us to log request and response details for debugging and monitoring purposes
// Note: Ensure that the logging system is properly initialized before using this client
func AddLoggingToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	// Wrap the existing transport with logging
	httpClient.Transport = logRoundTripper(httpClient.Transport)
	return nil
}

// AddDBStoreToHTTPClient wraps a http.Client's transporter with database storage functionality
// If the provided $httpClient is nil, an error is returned
// This allows us to store request and response details in a database for auditing and analysis purposes
// Note: Ensure that the database connection and storage system are properly initialized before using this client
func AddDBStoreToHTTPClient(httpClient *http.Client) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	if og == nil {
		return errors.New("cannot add DBStore transport until after go11y has been initialised")
	}

	if og.db == nil {
		return errors.New("go11y initialised with out a database - cannot add DBStore transport")
	}

	// Wrap the existing transport with logging
	httpClient.Transport = dbStoreRoundTripper(httpClient.Transport)

	return nil
}

// MetricsRecorder is a function type for recording metrics.
type MetricsRecorder func(statusCode int, method, path string, startTime time.Time)

// AddMetricsToHTTPClient wraps a http.Client's transporter with metrics recording functionality
// If the provided $httpClient is nil, an error is returned
// $recorder is the function that actually records the metrics - if it is nil an error is returned
// This allows us to record metrics for request and response details for monitoring purposes
func AddMetricsToHTTPClient(httpClient *http.Client, recorder MetricsRecorder, pathMaskFunc PathMask) (fault error) {
	if httpClient == nil {
		return errors.New("httpClient cannot be nil")
	}

	if recorder == nil {
		return errors.New("recorder cannot be nil")
	}

	// Wrap the existing transport with metrics recording
	httpClient.Transport = metricsRoundTripper(httpClient.Transport, recorder, pathMaskFunc)

	return nil
}
