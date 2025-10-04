package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client UniBarrage API 客户端
type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewClient 创建 API 客户端
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Response API 响应结构
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service 服务信息
type Service struct {
	Platform string `json:"platform"`
	RID      string `json:"rid"`
}

// StartServiceRequest 启动服务请求
type StartServiceRequest struct {
	RID    string `json:"rid"`
	Cookie string `json:"cookie,omitempty"`
}

// do 执行 HTTP 请求
func (c *Client) do(method, path string, body interface{}) (*Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp Response
	if err := json.Unmarshal(respData, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Code != 200 && apiResp.Code != 201 {
		return nil, fmt.Errorf("api error: code=%d, message=%s", apiResp.Code, apiResp.Message)
	}

	return &apiResp, nil
}

// Welcome 获取欢迎信息
func (c *Client) Welcome() (string, error) {
	resp, err := c.do("GET", "/api/v1/", nil)
	if err != nil {
		return "", err
	}
	return resp.Message, nil
}

// GetAllServices 获取所有服务状态
func (c *Client) GetAllServices() ([]Service, error) {
	resp, err := c.do("GET", "/api/v1/all", nil)
	if err != nil {
		return nil, err
	}

	var services []Service
	if err := json.Unmarshal(resp.Data, &services); err != nil {
		return nil, fmt.Errorf("unmarshal services: %w", err)
	}

	return services, nil
}

// GetPlatformServices 获取指定平台的所有服务
func (c *Client) GetPlatformServices(platform string) ([]Service, error) {
	resp, err := c.do("GET", fmt.Sprintf("/api/v1/%s", platform), nil)
	if err != nil {
		return nil, err
	}

	var services []Service
	if err := json.Unmarshal(resp.Data, &services); err != nil {
		return nil, fmt.Errorf("unmarshal services: %w", err)
	}

	return services, nil
}

// GetService 获取单个服务状态
func (c *Client) GetService(platform, rid string) (*Service, error) {
	resp, err := c.do("GET", fmt.Sprintf("/api/v1/%s/%s", platform, rid), nil)
	if err != nil {
		return nil, err
	}

	var service Service
	if err := json.Unmarshal(resp.Data, &service); err != nil {
		return nil, fmt.Errorf("unmarshal service: %w", err)
	}

	return &service, nil
}

// StartService 启动服务
func (c *Client) StartService(platform, rid, cookie string) (*Service, error) {
	req := StartServiceRequest{
		RID:    rid,
		Cookie: cookie,
	}

	resp, err := c.do("POST", fmt.Sprintf("/api/v1/%s", platform), req)
	if err != nil {
		return nil, err
	}

	var service Service
	if err := json.Unmarshal(resp.Data, &service); err != nil {
		return nil, fmt.Errorf("unmarshal service: %w", err)
	}

	return &service, nil
}

// StopService 停止服务
func (c *Client) StopService(platform, rid string) (*Service, error) {
	resp, err := c.do("DELETE", fmt.Sprintf("/api/v1/%s/%s", platform, rid), nil)
	if err != nil {
		return nil, err
	}

	var service Service
	if err := json.Unmarshal(resp.Data, &service); err != nil {
		return nil, fmt.Errorf("unmarshal service: %w", err)
	}

	return &service, nil
}
