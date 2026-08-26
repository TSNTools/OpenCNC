package uniclient

import (
	"OpenCNC/main_service/pkg/structures/configuration"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) AddStream(
	ctx context.Context,
	req *configuration.ConfigRequest,
) error {

	if req == nil {
		return fmt.Errorf("config request cannot be nil")
	}

	var body bytes.Buffer

	marshaler := &jsonpb.Marshaler{
		OrigName: false,
	}

	if err := marshaler.Marshal(&body, req); err != nil {
		return fmt.Errorf("failed to marshal UNI request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/add_stream",
		&body,
	)
	if err != nil {
		return fmt.Errorf("failed to create UNI request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send UNI request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"UNI server returned HTTP status %d",
			resp.StatusCode,
		)
	}

	return nil
}
