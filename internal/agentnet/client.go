package agentnet

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type CommandAck struct {
	Status string `json:"status"`
	Result any    `json:"result,omitempty"`
}

type HTTPError struct {
	Method     string
	Path       string
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return fmt.Sprintf("%s %s: http %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("%s %s: %s", e.Method, e.Path, e.Message)
}

func New(baseURL string, tlsConfig *tls.Config) (*Client, error) {
	return NewWithHTTPClient(baseURL, nil, tlsConfig)
}

func NewWithHTTPClient(baseURL string, httpClient *http.Client, tlsConfig *tls.Config) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		transport := &http.Transport{}
		if tlsConfig != nil {
			transport.TLSClientConfig = tlsConfig
		}
		httpClient = &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		}
	} else if tlsConfig != nil {
		if transport, ok := httpClient.Transport.(*http.Transport); ok && transport != nil {
			transport.TLSClientConfig = tlsConfig
		}
	}
	return &Client{
		baseURL:    strings.TrimRight(u.String(), "/"),
		httpClient: httpClient,
	}, nil
}

func (c *Client) Register(ctx context.Context, reqBody api.RegisterRequest) (api.RegisterResponse, error) {
	var out api.RegisterResponse
	if err := c.do(ctx, http.MethodPost, "/v1/nodes/register", reqBody, &out); err != nil {
		return api.RegisterResponse{}, err
	}
	return out, nil
}

func (c *Client) Report(ctx context.Context, nodeID string, reqBody api.NodeReport) (api.ReportAck, error) {
	var out api.ReportAck
	path := fmt.Sprintf("/v1/nodes/%s/report", url.PathEscape(nodeID))
	if err := c.do(ctx, http.MethodPost, path, reqBody, &out); err != nil {
		return api.ReportAck{}, err
	}
	return out, nil
}

func (c *Client) Commands(ctx context.Context, nodeID string) ([]api.Action, error) {
	var out []api.Action
	path := fmt.Sprintf("/v1/nodes/%s/commands", url.PathEscape(nodeID))
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Ack(ctx context.Context, nodeID, actionID string, ack CommandAck) error {
	path := fmt.Sprintf("/v1/nodes/%s/commands/%s/ack", url.PathEscape(nodeID), url.PathEscape(actionID))
	return c.do(ctx, http.MethodPost, path, ack, nil)
}

func (c *Client) do(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		data, _ := io.ReadAll(res.Body)
		return &HTTPError{
			Method:     method,
			Path:       path,
			StatusCode: res.StatusCode,
			Message:    strings.TrimSpace(string(data)),
		}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
