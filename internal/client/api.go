package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient UniBarrage API 客户端
type APIClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// APIClientConfig API 客户端配置
type APIClientConfig struct {
	Host      string
	Port      int
	AuthToken string
	Timeout   time.Duration
}

// NewAPIClient 创建新的 API 客户端
func NewAPIClient(config APIClientConfig) *APIClient {
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	return &APIClient{
		baseURL:   fmt.Sprintf("http://%s:%d/api/v1", config.Host, config.Port),
		authToken: config.AuthToken,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// APIResponse API 响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Platform string `json:"platform"`
	RID      string `json:"rid"`
}

// StartServiceRequest 启动服务请求
type StartServiceRequest struct {
	RID    string `json:"rid"`
	Cookie string `json:"cookie,omitempty"`
}

// doRequest 执行 HTTP 请求
func (c *APIClient) doRequest(method, path string, body interface{}) (*APIResponse, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &apiResp, nil
}

// Ping 测试 API 连接
func (c *APIClient) Ping() error {
	resp, err := c.doRequest("GET", "/", nil)
	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return fmt.Errorf("ping failed: %s", resp.Message)
	}

	return nil
}

// GetAllServices 获取所有服务状态
func (c *APIClient) GetAllServices() ([]ServiceInfo, error) {
	resp, err := c.doRequest("GET", "/all", nil)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("get all services failed: %s", resp.Message)
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	var services []ServiceInfo
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("unmarshal services: %w", err)
	}

	return services, nil
}

// GetPlatformServices 获取指定平台的所有服务
func (c *APIClient) GetPlatformServices(platform string) ([]ServiceInfo, error) {
	resp, err := c.doRequest("GET", "/"+platform, nil)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("get platform services failed: %s", resp.Message)
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	var services []ServiceInfo
	if err := json.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("unmarshal services: %w", err)
	}

	return services, nil
}

// GetService 获取指定服务状态
func (c *APIClient) GetService(platform, roomID string) (*ServiceInfo, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/%s/%s", platform, roomID), nil)
	if err != nil {
		return nil, err
	}

	if resp.Code != 200 {
		return nil, fmt.Errorf("get service failed: %s", resp.Message)
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	var service ServiceInfo
	if err := json.Unmarshal(data, &service); err != nil {
		return nil, fmt.Errorf("unmarshal service: %w", err)
	}

	return &service, nil
}

// StartService 启动服务
func (c *APIClient) StartService(platform, roomID, cookie string) error {
	reqBody := StartServiceRequest{
		RID:    roomID,
		Cookie: cookie,
	}

	resp, err := c.doRequest("POST", "/"+platform, reqBody)
	if err != nil {
		return err
	}

	if resp.Code != 201 && resp.Code != 200 {
		return fmt.Errorf("start service failed: %s", resp.Message)
	}

	return nil
}

// StopService 停止服务
func (c *APIClient) StopService(platform, roomID string) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/%s/%s", platform, roomID), nil)
	if err != nil {
		return err
	}

	if resp.Code != 200 {
		return fmt.Errorf("stop service failed: %s", resp.Message)
	}

	return nil
}
