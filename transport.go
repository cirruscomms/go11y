package go11y

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// RoundTripperFunc type is an adapter to allow the use of ordinary functions as http.RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls the RoundTripperFunc with the given request for each RoundTripper
func (rt RoundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	if rt == nil {
		return http.DefaultTransport.RoundTrip(r)
	}
	return rt(r)
}

func logRoundTripper(ctxWithObserver context.Context, next http.RoundTripper) http.RoundTripper {
	ctx, o, _ := Get(ctxWithObserver)
	return RoundTripperFunc(func(r *http.Request) (w *http.Response, fault error) {
		reqBody := []byte{}
		if r.Body != nil {
			defer func() {
				_ = r.Body.Close()
			}()
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			// Create a new request with the read body
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Use NopCloser to allow reading the body again if needed
		}

		requestArgs := []any{
			FieldRequestHeaders, RedactHeaders(r.Header),
			FieldRequestMethod, r.Method,
			FieldRequestURL, r.URL.String(),
			FieldRequestBody, reqBody,
		}

		o.log(ctx, 8, LevelInfo, "outbound call - request", requestArgs...)
		start := time.Now()

		// Send the actual request
		resp, err := next.RoundTrip(r)
		if err != nil {
			return nil, err
		}

		respBody := []byte{}
		// read the response body, use it to log the response body, then build a new response to return
		if resp.Body != nil {
			defer func() {
				_ = resp.Body.Close()
			}()

			respBody, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}
			// Create a new response with the read body
			resp.Body = io.NopCloser(bytes.NewBuffer(respBody)) // Use NopCloser to allow reading the body again if needed
		}

		duration := time.Since(start)

		responseArgs := []any{
			FieldCallDuration, duration,
			FieldStatusCode, resp.StatusCode,
			FieldResponseHeaders, RedactHeaders(resp.Header),
			FieldResponseBody, string(respBody),
		}
		o.log(ctx, 8, LevelInfo, "outbound call - response", responseArgs...)
		return resp, nil
	})
}

func dbStoreRoundTripper(ctxWithObserver context.Context, dbStorer DBStorer, next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (w *http.Response, fault error) {
		ctx, o, _ := Get(ctxWithObserver)
		reqBody := []byte{}
		if r.Body != nil {
			defer func() {
				_ = r.Body.Close()
			}()
			var err error
			reqBody, err = io.ReadAll(r.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read request body: %w", err)
			}
			// Create a new request with the read body
			r.Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Use NopCloser to allow reading the body again if needed
		}

		start := time.Now()

		resp, err := next.RoundTrip(r)
		if err != nil {
			return nil, err
		}

		respBody := []byte{}
		// read the response body, use it to log the response body, then build a new response to return
		if resp.Body != nil {
			defer func() {
				_ = resp.Body.Close()
			}()

			respBody, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}
			// Create a new response with the read body
			resp.Body = io.NopCloser(bytes.NewBuffer(respBody)) // Use NopCloser to allow reading the body again if needed
		}

		duration := time.Since(start)

		reqHeaders, err := json.Marshal(RedactHeaders(r.Header))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request headers: %w", err)
		}

		respHeaders, err := json.Marshal(RedactHeaders(resp.Header))
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response headers: %w", err)
		}

		dbStorer.SetURL(r.URL.String())
		dbStorer.SetMethod(r.Method)
		dbStorer.SetRequestHeaders(reqHeaders)
		dbStorer.SetRequestBody(pgtype.Text{String: string(reqBody), Valid: true})
		dbStorer.SetResponseTimeMS(duration.Milliseconds())
		dbStorer.SetResponseHeaders(respHeaders)
		dbStorer.SetResponseBody(pgtype.Text{String: string(respBody), Valid: true})
		dbStorer.SetStatusCode(int32(resp.StatusCode))
		err = dbStorer.Exec(ctx)
		if err != nil {
			o.Error("failed to store request/response in database", err, SeverityHigh)
			return nil, fmt.Errorf("failed to store request/response in database: %w", err)
		}

		return resp, nil
	})
}

func propagateRoundTripper(next http.RoundTripper) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (w *http.Response, fault error) {
		ctx := r.Context()

		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

		return next.RoundTrip(r)
	})
}

func metricsRoundTripper(next http.RoundTripper, recorder MetricsRecorder, pathMaskFunc PathMask) http.RoundTripper {
	return RoundTripperFunc(func(r *http.Request) (w *http.Response, fault error) {
		t0 := time.Now()

		resp, err := next.RoundTrip(r)

		path := r.URL.Path
		if pathMaskFunc != nil {
			path = pathMaskFunc(path)
		}

		recorder(resp.StatusCode, r.Method, path, t0)

		return resp, err
	})
}

// DBStorer interface defines methods for storing HTTP request and response details in a database
type DBStorer interface {
	SetURL(string)
	SetMethod(string)
	SetRequestHeaders([]byte)
	SetRequestBody(pgtype.Text)
	SetResponseTimeMS(int64)
	SetResponseHeaders([]byte)
	SetResponseBody(pgtype.Text)
	SetStatusCode(int32)
	Exec(ctx context.Context) error
}
