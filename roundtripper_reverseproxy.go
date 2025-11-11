package go11y

import (
	"errors"
	"net/http/httputil"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// AddTracingToReverseProxy wraps a httputil.ReverseProxy's transporter with OpenTelemetry instrumentation
// If the provided $reverseProxy is nil, it returns an error
// This allows us to capture request and response details in our telemetry data
// Note: Ensure that the OpenTelemetry SDK and otelhttp package are properly initialized before using this client
func AddTracingToReverseProxy(reverseProxy *httputil.ReverseProxy) (fault error) {
	if reverseProxy == nil {
		return errors.New("reverseProxy cannot be nil")
	}

	// Wrap the existing transport with OpenTelemetry tracing
	reverseProxy.Transport = otelhttp.NewTransport(reverseProxy.Transport)
	return nil
}

// AddPropagationToReverseProxy wraps a httputil.ReverseProxy's transporter with OpenTelemetry propagation
// If the provided $reverseProxy is nil, it returns an error
// This allows us to propagate tracing context across service boundaries
// Note: Ensure that the OpenTelemetry SDK and propagation package are properly initialized before using this client
func AddPropagationToReverseProxy(reverseProxy *httputil.ReverseProxy) (fault error) {
	if reverseProxy == nil {
		return errors.New("reverseProxy cannot be nil")
	}

	// Wrap the existing transport with OpenTelemetry tracing
	reverseProxy.Transport = propagateRoundTripper(reverseProxy.Transport)
	return nil
}

// AddLoggingToReverseProxy wraps a httputil.ReverseProxy's transporter with logging functionality
// If the provided $reverseProxy is nil, it returns an error
// This allows us to log request and response details for debugging and monitoring purposes
// Note: Ensure that the logging system is properly initialized before using this client
func AddLoggingToReverseProxy(reverseProxy *httputil.ReverseProxy) (fault error) {
	if reverseProxy == nil {
		return errors.New("reverseProxy cannot be nil")
	}

	// Wrap the existing transport with logging
	reverseProxy.Transport = logRoundTripper(reverseProxy.Transport)
	return nil
}

// AddDBStoreToReverseProxy wraps a httputil.ReverseProxy's transporter with database storage functionality
// If the provided $reverseProxy is nil, it returns an error
// This allows us to store request and response details in a database for auditing and analysis purposes
// Note: Ensure that the database connection and storage system are properly initialized before using this client
func AddDBStoreToReverseProxy(reverseProxy *httputil.ReverseProxy) (fault error) {
	if reverseProxy == nil {
		return errors.New("reverseProxy cannot be nil")
	}

	// Wrap the existing transport with logging
	reverseProxy.Transport = dbStoreRoundTripper(reverseProxy.Transport)
	return nil
}
