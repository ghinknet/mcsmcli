package mcsm

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"go.gh.ink/toolbox/xtype"
)

// ---- File Manager ----

// ListFiles fetches a file listing (GET /api/files/list).
func (c *Client) ListFiles(ctx context.Context, daemonID, uuid, target string, page, pageSize int) (*FilePage, RawMessage, error) {
	q := instQuery(daemonID, uuid)
	q.Set("target", target)
	q.Set("page", itoa(page))
	q.Set("page_size", itoa(pageSize))
	var out FilePage
	raw, err := c.Do(ctx, http.MethodGet, "/api/files/list", q, nil, &out)
	return &out, raw, err
}

// ReadFile gets file contents (PUT /api/files/, body contains only target).
func (c *Client) ReadFile(ctx context.Context, daemonID, uuid, target string) (string, error) {
	var out string
	_, err := c.Do(ctx, http.MethodPut, "/api/files/", instQuery(daemonID, uuid),
		xtype.H{"target": target}, &out)
	return out, err
}

// WriteFile updates file contents (PUT /api/files/, body contains target + text).
func (c *Client) WriteFile(ctx context.Context, daemonID, uuid, target, text string) error {
	_, err := c.Do(ctx, http.MethodPut, "/api/files/", instQuery(daemonID, uuid),
		xtype.H{"target": target, "text": text}, nil)
	return err
}

// CopyFiles copies files (POST /api/files/copy). targets is a list of [src, dst] pairs.
func (c *Client) CopyFiles(ctx context.Context, daemonID, uuid string, targets [][2]string) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/files/copy", instQuery(daemonID, uuid),
		xtype.H{"targets": targets}, nil)
	return err
}

// MoveFiles moves/renames files (PUT /api/files/move). targets is a list of [src, dst] pairs.
func (c *Client) MoveFiles(ctx context.Context, daemonID, uuid string, targets [][2]string) error {
	_, err := c.Do(ctx, http.MethodPut, "/api/files/move", instQuery(daemonID, uuid),
		xtype.H{"targets": targets}, nil)
	return err
}

// CompressFiles compresses files (POST /api/files/compress, type=1). code supports utf-8 only.
func (c *Client) CompressFiles(ctx context.Context, daemonID, uuid, source string, targets []string) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/files/compress", instQuery(daemonID, uuid),
		xtype.H{"type": 1, "code": "utf-8", "source": source, "targets": targets}, nil)
	return err
}

// DecompressFile decompresses a file (POST /api/files/compress, type=2). code supports utf-8/gbk/big5.
func (c *Client) DecompressFile(ctx context.Context, daemonID, uuid, source, dest, code string) error {
	if code == "" {
		code = "utf-8"
	}
	_, err := c.Do(ctx, http.MethodPost, "/api/files/compress", instQuery(daemonID, uuid),
		xtype.H{"type": 2, "code": code, "source": source, "targets": dest}, nil)
	return err
}

// DeleteFiles deletes files (DELETE /api/files).
func (c *Client) DeleteFiles(ctx context.Context, daemonID, uuid string, targets []string) error {
	_, err := c.Do(ctx, http.MethodDelete, "/api/files", instQuery(daemonID, uuid),
		xtype.H{"targets": targets}, nil)
	return err
}

// TouchFile creates an empty file (POST /api/files/touch).
func (c *Client) TouchFile(ctx context.Context, daemonID, uuid, target string) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/files/touch", instQuery(daemonID, uuid),
		xtype.H{"target": target}, nil)
	return err
}

// Mkdir creates a directory (POST /api/files/mkdir).
func (c *Client) Mkdir(ctx context.Context, daemonID, uuid, target string) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/files/mkdir", instQuery(daemonID, uuid),
		xtype.H{"target": target}, nil)
	return err
}

// DownloadFile is a two-stage download: request one-time credentials from the panel, then fetch the file from the daemon and write to localPath.
func (c *Client) DownloadFile(ctx context.Context, daemonID, uuid, remotePath, localPath string) error {
	q := instQuery(daemonID, uuid)
	q.Set("file_name", remotePath)
	var cred TransferCred
	if _, err := c.Do(ctx, http.MethodPost, "/api/files/download", q, nil, &cred); err != nil {
		return err
	}

	fileName := filepath.Base(strings.ReplaceAll(remotePath, "\\", "/"))
	scheme, host := c.transferEndpoint(cred.Addr)
	dlURL := fmt.Sprintf("%s://%s/download/%s/%s", scheme, host,
		url.PathEscape(cred.Password), url.PathEscape(fileName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return err
	}
	resp, err := transferHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("daemon download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write local file failed: %w", err)
	}
	return nil
}

// UploadFile is a two-stage upload: request one-time credentials from the panel, then push the file to the daemon as multipart.
func (c *Client) UploadFile(ctx context.Context, daemonID, uuid, uploadDir, localPath string) error {
	q := instQuery(daemonID, uuid)
	q.Set("upload_dir", uploadDir)
	var cred TransferCred
	if _, err := c.Do(ctx, http.MethodPost, "/api/files/upload", q, nil, &cred); err != nil {
		return err
	}

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("file", filepath.Base(localPath))
		if err == nil {
			_, err = io.Copy(part, f)
		}
		if err == nil {
			err = mw.Close()
		}
		pw.CloseWithError(err)
	}()

	upScheme, upHost := c.transferEndpoint(cred.Addr)
	upURL := fmt.Sprintf("%s://%s/upload/%s", upScheme, upHost, url.PathEscape(cred.Password))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upURL, pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := transferHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("daemon upload failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return fmt.Errorf("daemon upload failed: HTTP %d %s", resp.StatusCode, string(body))
	}
	return nil
}

// transferHTTP is used for direct file transfers with daemons. No timeout is set to avoid interrupting large transfers.
var transferHTTP = &http.Client{}

// transferEndpoint splits a transfer-credential address into an HTTP scheme and
// a bare host:port. Newer panels return addr with a WebSocket scheme
// (e.g. wss://host:port); older ones return bare host:port.
// Scheme priority: MCSM_DAEMON_SCHEME env > scheme embedded in addr > panel URL scheme.
func (c *Client) transferEndpoint(addr string) (scheme, host string) {
	host = addr
	if i := strings.Index(addr, "://"); i >= 0 {
		host = addr[i+3:]
		switch strings.ToLower(addr[:i]) {
		case "wss", "https":
			scheme = "https"
		case "ws", "http":
			scheme = "http"
		}
	}
	if s := os.Getenv("MCSM_DAEMON_SCHEME"); s != "" {
		scheme = s
	}
	if scheme == "" {
		if strings.HasPrefix(c.BaseURL, "https://") {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return scheme, host
}

// ---- Image Manager ----

// ListImages fetches the Docker image list (GET /api/environment/image).
func (c *Client) ListImages(ctx context.Context, daemonID string) (RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/environment/image", url.Values{"daemonId": {daemonID}}, nil, nil)
}

// ListContainers fetches the Docker container list (GET /api/environment/containers).
func (c *Client) ListContainers(ctx context.Context, daemonID string) (RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/environment/containers", url.Values{"daemonId": {daemonID}}, nil, nil)
}

// ListNetworks fetches the Docker network list (GET /api/environment/network).
func (c *Client) ListNetworks(ctx context.Context, daemonID string) (RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/environment/network", url.Values{"daemonId": {daemonID}}, nil, nil)
}

// BuildImage builds an image (POST /api/environment/image).
func (c *Client) BuildImage(ctx context.Context, daemonID, dockerFile, name, tag string) error {
	_, err := c.Do(ctx, http.MethodPost, "/api/environment/image", url.Values{"daemonId": {daemonID}},
		xtype.H{"dockerFile": dockerFile, "name": name, "tag": tag}, nil)
	return err
}

// BuildProgress queries image build progress (GET /api/environment/progress).
// Returns a map: image name -> -1 failed / 1 building / 2 complete.
func (c *Client) BuildProgress(ctx context.Context, daemonID string) (map[string]int, error) {
	var out map[string]int
	_, err := c.Do(ctx, http.MethodGet, "/api/environment/progress", url.Values{"daemonId": {daemonID}}, nil, &out)
	return out, err
}
