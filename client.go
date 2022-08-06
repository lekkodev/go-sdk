package client

import (
	"context"
	"log"
	"net/http"

	"github.com/lekkodev/core/pkg/gen/proto/go-connect/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "github.com/lekkodev/core/pkg/gen/proto/go/lekko/backend/v1beta1"

	"github.com/bufbuild/connect-go"
)

type Client struct {
	orgName, namespace string
	lekkoClient        backendv1beta1connect.ConfigurationServiceClient
}

func NewClient(orgName, namespace string) *Client {
	return &Client{
		orgName:     orgName,
		namespace:   namespace,
		lekkoClient: backendv1beta1connect.NewConfigurationServiceClient(http.DefaultClient, "http://localhost:8080/"),
	}
}

func (c *Client) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	resp, err := c.lekkoClient.GetBoolValue(ctx, connect.NewRequest(&backendv1beta1.GetBoolValueRequest{
		Key:       key,
		Namespace: c.namespace,
	}))
	if err != nil {
		log.Printf("error hitting lekko backend: %v\n", err)
		return defaultValue
	}
	return resp.Msg.GetValue()
}
