package mcsm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDoAuthAndEnvelope 验证 apikey 注入、必备请求头与 envelope 解析。
func TestDoAuthAndEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") != "test-key" {
			t.Errorf("apikey 未注入: %s", r.URL.RawQuery)
		}
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			t.Errorf("缺少 X-Requested-With 请求头")
		}
		switch r.URL.Path {
		case "/api/protected_instance/open":
			w.Write([]byte(`{"status":200,"data":{"instanceUuid":"abc"},"time":1}`))
		case "/api/overview":
			w.Write([]byte(`{"status":403,"data":"Permission denied","time":1}`))
		case "/api/instance":
			w.Write([]byte(`{"status":200,"data":{"instanceUuid":"abc","status":3,"config":{"nickname":"srv"}},"time":1}`))
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key", 5*time.Second)
	ctx := context.Background()

	if err := c.InstancePower(ctx, "d1", "abc", "open"); err != nil {
		t.Fatalf("InstancePower: %v", err)
	}

	_, _, err := c.GetOverview(ctx)
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Status != 403 || apiErr.Message != "Permission denied" {
		t.Fatalf("期望 403 APIError，实际: %v", err)
	}

	ins, _, err := c.GetInstance(ctx, "d1", "abc")
	if err != nil {
		t.Fatalf("GetInstance: %v", err)
	}
	if ins.Config.Nickname != "srv" || StatusText(ins.Status) != "running" {
		t.Fatalf("实例解析错误: %+v", ins)
	}
}
