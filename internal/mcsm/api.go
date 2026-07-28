package mcsm

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"go.gh.ink/toolbox/xtype"
)

// ---- Dashboard ----

// GetOverview fetches the panel overview (GET /api/overview).
func (c *Client) GetOverview(ctx context.Context) (*Overview, RawMessage, error) {
	var out Overview
	raw, err := c.Do(ctx, http.MethodGet, "/api/overview", nil, nil, &out)
	return &out, raw, err
}

// ---- Daemon ----

// ListDaemonsSystem fetches daemon system info (GET /api/service/remote_services_system).
func (c *Client) ListDaemonsSystem(ctx context.Context) (RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/service/remote_services_system", nil, nil, nil)
}

// AddDaemon adds a daemon (POST /api/service/remote_service). Returns the new daemon ID.
func (c *Client) AddDaemon(ctx context.Context, ip string, port int, prefix, remarks, apiKey string) (string, error) {
	var id string
	_, err := c.Do(ctx, http.MethodPost, "/api/service/remote_service", nil, xtype.H{
		"ip": ip, "port": port, "prefix": prefix, "remarks": remarks, "apiKey": apiKey,
	}, &id)
	return id, err
}

// DeleteDaemon deletes a daemon (DELETE /api/service/remote_service).
func (c *Client) DeleteDaemon(ctx context.Context, daemonID string) error {
	_, err := c.Do(ctx, http.MethodDelete, "/api/service/remote_service",
		url.Values{"uuid": {daemonID}}, nil, nil)
	return err
}

// LinkDaemon attempts to reconnect a daemon (GET /api/service/link_remote_service).
func (c *Client) LinkDaemon(ctx context.Context, daemonID string) error {
	_, err := c.Do(ctx, http.MethodGet, "/api/service/link_remote_service",
		url.Values{"uuid": {daemonID}}, nil, nil)
	return err
}

// UpdateDaemon updates daemon connection config (PUT /api/service/remote_service).
// body must include uuid/ip/port/prefix/available/remarks/apiKey.
func (c *Client) UpdateDaemon(ctx context.Context, daemonID string, body xtype.H) error {
	_, err := c.Do(ctx, http.MethodPut, "/api/service/remote_service",
		url.Values{"uuid": {daemonID}}, body, nil)
	return err
}

// ---- Instance ----

// ListInstances fetches the instance list for a daemon (GET /api/service/remote_service_instances).
func (c *Client) ListInstances(ctx context.Context, daemonID string, page, pageSize int, name, status string) (*InstancePage, RawMessage, error) {
	q := url.Values{
		"daemonId":  {daemonID},
		"page":      {itoa(page)},
		"page_size": {itoa(pageSize)},
		"status":    {status},
	}
	if name != "" {
		q.Set("instance_name", name)
	}
	var out InstancePage
	raw, err := c.Do(ctx, http.MethodGet, "/api/service/remote_service_instances", q, nil, &out)
	return &out, raw, err
}

// GetInstance fetches instance details (GET /api/instance).
func (c *Client) GetInstance(ctx context.Context, daemonID, uuid string) (*InstanceDetail, RawMessage, error) {
	var out InstanceDetail
	raw, err := c.Do(ctx, http.MethodGet, "/api/instance", instQuery(daemonID, uuid), nil, &out)
	return &out, raw, err
}

// CreateInstance creates an instance (POST /api/instance).
func (c *Client) CreateInstance(ctx context.Context, daemonID string, cfg any) (RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/instance", url.Values{"daemonId": {daemonID}}, cfg, nil)
}

// UpdateInstanceConfig updates instance config (PUT /api/instance).
func (c *Client) UpdateInstanceConfig(ctx context.Context, daemonID, uuid string, cfg any) (RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/instance", instQuery(daemonID, uuid), cfg, nil)
}

// DeleteInstances deletes instances (DELETE /api/instance).
func (c *Client) DeleteInstances(ctx context.Context, daemonID string, uuids []string, deleteFile bool) (RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/instance", url.Values{"daemonId": {daemonID}},
		xtype.H{"uuids": uuids, "deleteFile": deleteFile}, nil)
}

// InstancePower performs a power operation: open/stop/restart/kill
// （GET /api/protected_instance/{op}）。
func (c *Client) InstancePower(ctx context.Context, daemonID, uuid, op string) error {
	_, err := c.Do(ctx, http.MethodGet, "/api/protected_instance/"+op, instQuery(daemonID, uuid), nil, nil)
	return err
}

// BatchInstanceOp performs a batch operation: start/stop/restart/kill
// (POST /api/instance/multi_{op}, body is an InstanceRef array).
func (c *Client) BatchInstanceOp(ctx context.Context, op string, refs []InstanceRef) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/instance/multi_"+op, nil, refs, nil)
	return err
}

// UpgradeInstance triggers the instance update task (POST /api/protected_instance/asynchronous, task_name=update).
func (c *Client) UpgradeInstance(ctx context.Context, daemonID, uuid string) error {
	q := instQuery(daemonID, uuid)
	q.Set("task_name", "update")
	_, err := c.Do(ctx, http.MethodPost, "/api/protected_instance/asynchronous", q, nil, nil)
	return err
}

// SendCommand sends a command to an instance (GET /api/protected_instance/command).
func (c *Client) SendCommand(ctx context.Context, daemonID, uuid, command string) error {
	q := instQuery(daemonID, uuid)
	q.Set("command", command)
	_, err := c.Do(ctx, http.MethodGet, "/api/protected_instance/command", q, nil, nil)
	return err
}

// GetOutputLog fetches the instance terminal output (GET /api/protected_instance/outputlog).
// sizeKB of 0 returns all logs.
func (c *Client) GetOutputLog(ctx context.Context, daemonID, uuid string, sizeKB int) (string, error) {
	q := instQuery(daemonID, uuid)
	if sizeKB > 0 {
		q.Set("size", itoa(sizeKB))
	}
	var out string
	_, err := c.Do(ctx, http.MethodGet, "/api/protected_instance/outputlog", q, nil, &out)
	return out, err
}

// ReinstallInstance reinstalls an instance (POST /api/protected_instance/install_instance).
func (c *Client) ReinstallInstance(ctx context.Context, daemonID, uuid, targetURL, title, description string) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/protected_instance/install_instance",
		instQuery(daemonID, uuid),
		xtype.H{"targetUrl": targetURL, "title": title, "description": description}, nil)
	return err
}

// ---- Users ----

// ListUsers searches for users (GET /api/auth/search).
func (c *Client) ListUsers(ctx context.Context, userName, role string, page, pageSize int) (*UserPage, RawMessage, error) {
	q := url.Values{"page": {itoa(page)}, "page_size": {itoa(pageSize)}}
	if userName != "" {
		q.Set("userName", userName)
	}
	if role != "" {
		q.Set("role", role)
	}
	var out UserPage
	raw, err := c.Do(ctx, http.MethodGet, "/api/auth/search", q, nil, &out)
	return &out, raw, err
}

// CreateUser creates a user (POST /api/auth).
func (c *Client) CreateUser(ctx context.Context, username, password string, permission int) (RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/auth", nil,
		xtype.H{"username": username, "password": password, "permission": permission}, nil)
}

// UpdateUser updates a user (PUT /api/auth). The config is the full user object.
func (c *Client) UpdateUser(ctx context.Context, uuid string, cfg any) error {
	_, err := c.Do(ctx, http.MethodPut, "/api/auth", nil, xtype.H{"uuid": uuid, "config": cfg}, nil)
	return err
}

// DeleteUsers deletes users (DELETE /api/auth, body is a uuid array).
func (c *Client) DeleteUsers(ctx context.Context, uuids []string) error {
	_, err := c.Do(ctx, http.MethodDelete, "/api/auth", nil, uuids, nil)
	return err
}

// ---- helpers ----

func instQuery(daemonID, uuid string) url.Values {
	return url.Values{"daemonId": {daemonID}, "uuid": {uuid}}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
