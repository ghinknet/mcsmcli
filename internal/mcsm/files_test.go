package mcsm

import (
	"testing"
	"time"
)

// TestTransferEndpoint 验证传输凭据地址的协议归一化：
// 新版面板返回的 addr 携带 ws/wss 协议前缀，旧版返回裸 host:port。
func TestTransferEndpoint(t *testing.T) {
	httpsClient := New("https://panel.example.com", "k", 5*time.Second)
	httpClient := New("http://panel.example.com", "k", 5*time.Second)

	cases := []struct {
		name       string
		client     *Client
		addr       string
		env        string
		wantScheme string
		wantHost   string
	}{
		{"裸地址+https面板", httpsClient, "node1:24444", "", "https", "node1:24444"},
		{"裸地址+http面板", httpClient, "node1:24444", "", "http", "node1:24444"},
		{"wss前缀", httpClient, "wss://node1:15423", "", "https", "node1:15423"},
		{"ws前缀", httpsClient, "ws://node1:15423", "", "http", "node1:15423"},
		{"https前缀", httpClient, "https://node1:15423", "", "https", "node1:15423"},
		{"http前缀", httpsClient, "http://node1:15423", "", "http", "node1:15423"},
		{"env覆盖addr前缀", httpsClient, "wss://node1:15423", "http", "http", "node1:15423"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("MCSM_DAEMON_SCHEME", tc.env)
			}
			scheme, host := tc.client.transferEndpoint(tc.addr)
			if scheme != tc.wantScheme || host != tc.wantHost {
				t.Errorf("transferEndpoint(%q) = (%q, %q), 期望 (%q, %q)",
					tc.addr, scheme, host, tc.wantScheme, tc.wantHost)
			}
		})
	}
}
