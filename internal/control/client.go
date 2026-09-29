package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"prochub/internal/process"
)

// Client talks to a running application's control server. Build it with
// NewClient, which reads the auth file written by the application.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient reads auth.json from dataDir and returns a client for the running
// application. A missing auth file means the application is not running.
func NewClient(dataDir string) (*Client, error) {
	info, err := ReadAuth(dataDir)
	if err != nil {
		return nil, err
	}
	if info.Port <= 0 || info.Token == "" {
		return nil, errors.New("auth.json 内容无效，请重启 ProcHub")
	}
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", info.Port),
		token:   info.Token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Status returns the application runtime status.
func (c *Client) Status() (StatusInfo, error) {
	var out StatusInfo
	err := c.do(http.MethodGet, "/control/status", nil, &out)
	return out, err
}

// ListProcesses returns every registered process with its status.
func (c *Client) ListProcesses() ([]process.Snapshot, error) {
	var out []process.Snapshot
	err := c.do(http.MethodGet, "/control/processes", nil, &out)
	return out, err
}

// StartProcess starts a process by ID.
func (c *Client) StartProcess(id string) error {
	return c.do(http.MethodPost, "/control/processes/"+url.PathEscape(id)+"/start", nil, nil)
}

// StopProcess stops a process by ID.
func (c *Client) StopProcess(id string) error {
	return c.do(http.MethodPost, "/control/processes/"+url.PathEscape(id)+"/stop", nil, nil)
}

// Logs returns up to the last tail lines of a process's log file.
func (c *Client) Logs(id string, tail int) ([]string, error) {
	var out struct {
		Lines []string `json:"lines"`
	}
	path := fmt.Sprintf("/control/processes/%s/logs?tail=%d", url.PathEscape(id), tail)
	if err := c.do(http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Lines, nil
}

// Config returns the current settings summary.
func (c *Client) Config() (ConfigSummary, error) {
	var out ConfigSummary
	err := c.do(http.MethodGet, "/control/config", nil, &out)
	return out, err
}

// UpdateConfig applies a settings patch.
func (c *Client) UpdateConfig(patch ConfigPatch) error {
	return c.do(http.MethodPost, "/control/config", patch, nil)
}

func (c *Client) do(method, path string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set(TokenHeader, c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("连接 ProcHub 失败：%w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		OK    bool            `json:"ok"`
		Data  json.RawMessage `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("响应解析失败（HTTP %d）：%w", resp.StatusCode, err)
	}
	if !payload.OK {
		if payload.Error != "" {
			return errors.New(payload.Error)
		}
		return fmt.Errorf("请求失败（HTTP %d）", resp.StatusCode)
	}
	if out != nil && len(payload.Data) > 0 {
		return json.Unmarshal(payload.Data, out)
	}
	return nil
}
