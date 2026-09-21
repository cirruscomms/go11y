package go11y_test

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/cirruscomms/go11y"
	provClient "github.com/cirruscomms/provisioning-service/v3/client/v1"
)

func provSvcClientWithGo11yTransport(ctxWithObserver context.Context, timeout time.Duration) provClient.ClientOption {
	return func(c *provClient.Client) error {
		httpClient := go11y.HTTPClient{
			Client: &http.Client{
				Transport: http.DefaultTransport,
				Timeout:   timeout,
			},
		}

		err := httpClient.AddRequestID("go11y-service")
		if err != nil {
			return err
		}
		// Other transport methods can be called here

		c.Client = httpClient
		return nil
	}
}

func ExampleHTTPClient_AddRequestID() {
	ctx := context.Background()

	cfg := go11y.Configuration{}

	ctx, _, err := go11y.Initialise(ctx, &cfg, os.Stdout, os.Stderr)
	if err != nil {
		panic(err)
	}

	provClient, err := provClient.NewClientWithResponses("http://localhost:8080", provSvcClientWithGo11yTransport(ctx, 30*time.Second))
	if err != nil {
		panic(err)
	}

	orderID := 123
	_, err = provClient.GetOrderDetailsV1WithResponse(ctx, orderID)
	if err != nil {
		panic(err)
	}
}
