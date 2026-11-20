package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"navbox/internal/dto"
)

// SafeKeepBackupConfig 定义 safekeep_backup 定时任务的持久化与 DTO 配置。
type SafeKeepBackupConfig struct {
	APIBaseURL      string `json:"api_base_url"`
	APIKey          string `json:"api_key,omitempty"`
	APIKeyEncrypted string `json:"api_key_encrypted,omitempty"`
	HasAPIKey       bool   `json:"has_api_key,omitempty"`
	Targets         string `json:"targets"`
	Path            string `json:"path"`
	Mode            string `json:"mode"`
	Rename          string `json:"rename"`
	SkipDedup       bool   `json:"skip_dedup"`
	SkipTLSVerify   bool   `json:"skip_tls_verify"`
}

func defaultSafeKeepBackupConfig() SafeKeepBackupConfig {
	return SafeKeepBackupConfig{
		Path: "navbox-backups",
		Mode: "sync",
	}
}

// ParseSafeKeepBackupConfig 解析存储的配置 JSON 并填充安全默认值。
func ParseSafeKeepBackupConfig(raw string) SafeKeepBackupConfig {
	cfg := defaultSafeKeepBackupConfig()
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.Mode == "" {
		cfg.Mode = "sync"
	}
	return cfg
}

// SanitizeSafeKeepBackupConfigJSON 移除加密与明文密钥并补充 has_api_key 状态标记。
func SanitizeSafeKeepBackupConfigJSON(raw string) string {
	cfg := ParseSafeKeepBackupConfig(raw)
	cfg.HasAPIKey = cfg.APIKeyEncrypted != "" || cfg.APIKey != ""
	cfg.APIKey = ""
	cfg.APIKeyEncrypted = ""
	b, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func isMaskedSecretPlaceholder(v string) bool {
	return v == "*****" || v == "••••••"
}

// MergeSafeKeepBackupConfig 校验输入配置并安全保留或加密保存 API Key。
func MergeSafeKeepBackupConfig(cipher SecretCipher, existingRaw string, incoming json.RawMessage) (string, error) {
	if cipher == nil {
		return "", errors.New("secret cipher is required")
	}
	existing := ParseSafeKeepBackupConfig(existingRaw)

	var req struct {
		APIBaseURL    string  `json:"api_base_url"`
		APIKey        *string `json:"api_key"`
		Targets       string  `json:"targets"`
		Path          string  `json:"path"`
		Mode          string  `json:"mode"`
		Rename        string  `json:"rename"`
		SkipDedup     *bool   `json:"skip_dedup"`
		SkipTLSVerify *bool   `json:"skip_tls_verify"`
	}
	if len(incoming) > 0 {
		if err := json.Unmarshal(incoming, &req); err != nil {
			return "", newValidationError(fmt.Sprintf("SafeKeep 备份配置格式无效：%v", err))
		}
	}

	baseURL, err := normalizeSafeKeepBaseURL(req.APIBaseURL)
	if err != nil {
		return "", err
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "sync"
	}
	if mode != "sync" && mode != "async" {
		return "", newValidationError("mode 仅支持 sync 或 async")
	}

	rename := strings.TrimSpace(req.Rename)
	if rename == "." || rename == ".." || rename == "/" || strings.ContainsAny(rename, `/\`) {
		return "", newValidationError("rename 文件名不合法")
	}

	apiKeyEnc := existing.APIKeyEncrypted
	if req.APIKey != nil {
		val := strings.TrimSpace(*req.APIKey)
		if val == "" {
			apiKeyEnc = ""
		} else if !isMaskedSecretPlaceholder(val) {
			enc, err := cipher.EncryptString(val)
			if err != nil {
				return "", fmt.Errorf("encrypt safekeep api_key: %w", err)
			}
			apiKeyEnc = enc
		}
	}

	skipDedup := existing.SkipDedup
	if req.SkipDedup != nil {
		skipDedup = *req.SkipDedup
	}
	skipTLSVerify := existing.SkipTLSVerify
	if req.SkipTLSVerify != nil {
		skipTLSVerify = *req.SkipTLSVerify
	}

	out := SafeKeepBackupConfig{
		APIBaseURL:      baseURL,
		APIKeyEncrypted: apiKeyEnc,
		Targets:         strings.TrimSpace(req.Targets),
		Path:            strings.TrimSpace(req.Path),
		Mode:            mode,
		Rename:          rename,
		SkipDedup:       skipDedup,
		SkipTLSVerify:   skipTLSVerify,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RevealSafeKeepBackupAPIKey 解密存储的 SafeKeep API Key。
func RevealSafeKeepBackupAPIKey(cipher SecretCipher, raw string) (string, error) {
	if cipher == nil {
		return "", errors.New("secret cipher is required")
	}
	cfg := ParseSafeKeepBackupConfig(raw)
	if cfg.APIKeyEncrypted == "" {
		return "", nil
	}
	apiKey, err := cipher.DecryptString(cfg.APIKeyEncrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt safekeep api_key: %w", err)
	}
	return apiKey, nil
}

func normalizeSafeKeepBaseURL(raw string) (string, error) {
	baseURL := strings.TrimSpace(raw)
	if baseURL == "" {
		return "", nil
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", newValidationError("api_base_url 必须为合法的 http 或 https 地址")
	}
	return strings.TrimRight(baseURL, "/"), nil
}

type SafeKeepUploadRequest struct {
	APIBaseURL    string
	APIKey        string
	FileName      string
	Content       []byte
	Mode          string
	Path          string
	Targets       string
	Rename        string
	SkipDedup     bool
	SkipTLSVerify bool
	Metadata      map[string]any
}

type SafeKeepUploadResult struct {
	Endpoint        string
	TaskID          string
	Status          string
	StoredFilename  string
	Deduplicated    bool
	TimedOutToAsync bool
}

// SafeKeepClient 封装对 SafeKeep 客户端数据面 HTTP API 的调用。
type SafeKeepClient struct {
	httpClient *http.Client
}

func NewSafeKeepClient() *SafeKeepClient {
	return &SafeKeepClient{}
}

func (c *SafeKeepClient) clientFor(skipTLSVerify bool, defaultTimeout time.Duration) *http.Client {
	if c != nil && c.httpClient != nil && !skipTLSVerify {
		return c.httpClient
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if skipTLSVerify {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true //nolint:gosec // 用户显式开启自签证书跳过校验
	}
	return &http.Client{
		Transport: transport,
		Timeout:   defaultTimeout,
	}
}

// UploadBackup 调用 SafeKeep POST /api/v1/backups 上传导出压缩包。
func (c *SafeKeepClient) UploadBackup(ctx context.Context, req SafeKeepUploadRequest) (SafeKeepUploadResult, error) {
	baseURL, err := normalizeSafeKeepBaseURL(req.APIBaseURL)
	if err != nil {
		return SafeKeepUploadResult{}, err
	}
	if baseURL == "" {
		return SafeKeepUploadResult{}, errors.New("safekeep api_base_url is not configured")
	}
	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return SafeKeepUploadResult{}, errors.New("safekeep api_key is not configured")
	}

	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = "sync"
	}

	var bodyBuf bytes.Buffer
	mw := multipart.NewWriter(&bodyBuf)
	_ = mw.WriteField("mode", mode)
	if strings.TrimSpace(req.Path) != "" {
		_ = mw.WriteField("path", strings.TrimSpace(req.Path))
	}
	if strings.TrimSpace(req.Targets) != "" {
		_ = mw.WriteField("targets", strings.TrimSpace(req.Targets))
	}
	if strings.TrimSpace(req.Rename) != "" {
		_ = mw.WriteField("rename", strings.TrimSpace(req.Rename))
	}
	if req.SkipDedup {
		_ = mw.WriteField("dedup", "false")
	}
	if len(req.Metadata) > 0 {
		if metaBytes, mErr := json.Marshal(req.Metadata); mErr == nil {
			_ = mw.WriteField("metadata", string(metaBytes))
		}
	}

	fileName := strings.TrimSpace(req.FileName)
	if fileName == "" {
		fileName = fmt.Sprintf("navbox-export-%s.zip", time.Now().UTC().Format("20060102150405"))
	}
	fw, err := mw.CreateFormFile("file", fileName)
	if err != nil {
		return SafeKeepUploadResult{}, fmt.Errorf("create multipart form file: %w", err)
	}
	if _, err := fw.Write(req.Content); err != nil {
		return SafeKeepUploadResult{}, fmt.Errorf("write backup zip to multipart body: %w", err)
	}
	if err := mw.Close(); err != nil {
		return SafeKeepUploadResult{}, fmt.Errorf("close multipart writer: %w", err)
	}

	endpoint := baseURL + "/api/v1/backups"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBuf.Bytes()))
	if err != nil {
		return SafeKeepUploadResult{}, fmt.Errorf("build safekeep backup request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	httpReq.Header.Set("X-API-Key", apiKey)

	resp, err := c.clientFor(req.SkipTLSVerify, 0).Do(httpReq)
	if err != nil {
		return SafeKeepUploadResult{}, fmt.Errorf("POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		var out struct {
			TimedOutToAsync bool `json:"timed_out_to_async"`
			Task            struct {
				ID               string `json:"id"`
				Status           string `json:"status"`
				OriginalFilename string `json:"original_filename"`
				RemoteFilename   string `json:"remote_filename"`
				Deduplicated     bool   `json:"deduplicated"`
			} `json:"task"`
		}
		_ = json.Unmarshal(respBytes, &out)
		storedName := out.Task.RemoteFilename
		if storedName == "" {
			if strings.TrimSpace(req.Rename) != "" {
				storedName = strings.TrimSpace(req.Rename)
			} else {
				storedName = fileName
			}
		}
		status := out.Task.Status
		if resp.StatusCode == http.StatusAccepted && out.TimedOutToAsync {
			status = "timed_out_to_async"
		}
		if status == "" {
			status = "accepted"
		}
		return SafeKeepUploadResult{
			Endpoint:        endpoint,
			TaskID:          out.Task.ID,
			Status:          status,
			StoredFilename:  storedName,
			Deduplicated:    out.Task.Deduplicated,
			TimedOutToAsync: out.TimedOutToAsync,
		}, nil
	}

	return SafeKeepUploadResult{}, parseSafeKeepError("backup API", resp.StatusCode, respBytes)
}

// ListUpstreams 调用 SafeKeep GET /api/v1/upstreams 验证连通性与 API Key，并返回可用上游列表。
func (c *SafeKeepClient) ListUpstreams(ctx context.Context, rawBaseURL string, apiKey string, skipTLSVerify bool) ([]dto.SafeKeepUpstreamResp, error) {
	baseURL, err := normalizeSafeKeepBaseURL(rawBaseURL)
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		return nil, newValidationError("请填写 SafeKeep API 地址")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, newValidationError("请填写或先保存 SafeKeep API Key")
	}

	endpoint := baseURL + "/api/v1/upstreams"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build safekeep upstreams request: %w", err)
	}
	httpReq.Header.Set("X-API-Key", apiKey)

	resp, err := c.clientFor(skipTLSVerify, 15*time.Second).Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, parseSafeKeepError("upstreams API", resp.StatusCode, respBytes)
	}

	var out struct {
		Items []dto.SafeKeepUpstreamResp `json:"items"`
	}
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, fmt.Errorf("decode safekeep upstreams response: %w", err)
	}
	if out.Items == nil {
		out.Items = []dto.SafeKeepUpstreamResp{}
	}
	return out.Items, nil
}

func parseSafeKeepError(apiName string, statusCode int, respBytes []byte) error {
	var errOut struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Task struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			ErrorSummary string `json:"error_summary"`
		} `json:"task"`
	}
	_ = json.Unmarshal(respBytes, &errOut)
	if errOut.Task.ID != "" {
		return fmt.Errorf("SafeKeep %s %d (task %s, %s): %s", apiName, statusCode, errOut.Task.ID, errOut.Task.Status, errOut.Task.ErrorSummary)
	}
	if errOut.Error.Message != "" {
		return fmt.Errorf("SafeKeep %s %d (%s): %s", apiName, statusCode, errOut.Error.Code, errOut.Error.Message)
	}
	return fmt.Errorf("SafeKeep %s returned HTTP %d: %s", apiName, statusCode, strings.TrimSpace(string(respBytes)))
}
