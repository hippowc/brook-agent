package networktool

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"brook-agent/internal/core/tool"
)

const Name = "network"

// Tool 提供 HTTP 请求能力。
type Tool struct {
	client *http.Client
}

func init() {
	tool.Register(Name, func() tool.Tool {
		return &Tool{
			client: &http.Client{Timeout: 30 * time.Second},
		}
	})
}

func (t *Tool) Name() string { return Name }

// Execute 支持 get/post 两类网络请求。
func (t *Tool) Execute(ctx context.Context, call tool.Call) (tool.Result, error) {
	method := strings.ToUpper(call.Args["method"])
	url := call.Args["url"]
	body := call.Args["body"]

	switch method {
	case "GET", "":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		resp, err := t.client.Do(req)
		if err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return tool.Result{Output: string(data)}, nil
	case "POST":
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
		if err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		resp, err := t.client.Do(req)
		if err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return tool.Result{Output: string(data)}, nil
	default:
		return tool.Result{IsError: true, Output: "unsupported method"}, fmt.Errorf("unsupported method: %s", method)
	}
}
