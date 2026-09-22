package go11y_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	apiClient "github.com/cirruscomms/go-template-service-api/client"
	"github.com/cirruscomms/go11y"
)

func provSvcClientWithGo11yTransport(ctxWithObserver context.Context, timeout time.Duration) apiClient.ClientOption {
	return func(c *apiClient.Client) error {
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

	templateClient, err := apiClient.NewClientWithResponses("http://localhost:8080", provSvcClientWithGo11yTransport(ctx, 30*time.Second))
	if err != nil {
		panic(err)
	}

	widgetID := fmt.Sprint(rune(123))
	resp, err := templateClient.WidgetGet(ctx, widgetID)
	if err != nil {
		panic(err)
	}

	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	fmt.Println("Response status:", resp.StatusCode)
}
