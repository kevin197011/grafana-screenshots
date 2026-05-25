// Grafana Dashboard 截图：agentless HTTP /render API + Lark 群推送 + HTTP 触发与可选定时调度。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "time/tzdata" // 嵌入时区库，运行镜像无需安装 tzdata

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

const (
	defaultScreenshotDir           = "/data/screenshots"
	defaultScreenshotRetentionDays = 30
	defaultRenderW                 = 1920
	defaultRenderH                 = 2400
	defaultRenderTimeout           = 120
	defaultRenderTimeoutFullPage   = 180
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "scheduler":
			runScheduler()
			return
		case "server":
			runServer()
			return
		case "once", "run":
			os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		}
	}
	if err := runOnceFromFlags(); err != nil {
		fatal(err)
	}
}

var (
	triggerMu      sync.Mutex
	triggerRunning bool
)

func runServer() {
	loadEnv()

	addr := envOr("LISTEN_ADDR", ":8111")
	if envBool("ENABLE_SCHEDULER", false) {
		go runScheduler()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/trigger", handleTrigger)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handleTrigger(w, r)
	})

	fmt.Printf("HTTP 监听 %s\n", addr)
	fmt.Println("  GET /health   健康检查")
	fmt.Println("  GET /trigger  截图并发送到 Lark（?dry-run=1 仅截图）")
	if secret := strings.TrimSpace(os.Getenv("TRIGGER_SECRET")); secret != "" {
		fmt.Println("  需带 ?token=<TRIGGER_SECRET>")
	}
	if err := http.ListenAndServe(addr, mux); err != nil {
		fatal(err)
	}
}

func handleTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
		return
	}
	if secret := strings.TrimSpace(os.Getenv("TRIGGER_SECRET")); secret != "" {
		if r.URL.Query().Get("token") != secret {
			http.Error(w, "unauthorized\n", http.StatusUnauthorized)
			return
		}
	}

	triggerMu.Lock()
	if triggerRunning {
		triggerMu.Unlock()
		http.Error(w, "job already running\n", http.StatusConflict)
		return
	}
	triggerRunning = true
	triggerMu.Unlock()
	defer func() {
		triggerMu.Lock()
		triggerRunning = false
		triggerMu.Unlock()
	}()

	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dry-run")))
	dryRun := q == "1" || q == "true" || q == "yes"

	if err := runOnce(dryRun); err != nil {
		http.Error(w, err.Error()+"\n", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func runOnceFromFlags() error {
	dryRun := flag.Bool("dry-run", false, "只截图，不发送")
	flag.Parse()
	return runOnce(*dryRun)
}

func loadEnv() {
	// 优先 grafana.env：Compose 不会自动解析该文件名。
	// godotenv 会把 "$__all" 里的 "$_" 当成变量展开成 "all"，故 GRAFANA_URL 需从文件原文恢复。
	for _, p := range []string{"grafana.env", "/app/.env", ".env"} {
		if err := godotenv.Load(p); err == nil {
			restoreGrafanaURLFromFile(p)
			return
		}
	}
}

func restoreGrafanaURLFromFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "GRAFANA_URL" {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		if unquoted, ok := unquoteEnvValue(val); ok {
			val = unquoted
		}
		_ = os.Setenv("GRAFANA_URL", val)
		return
	}
}

func unquoteEnvValue(s string) (string, bool) {
	if len(s) < 2 {
		return s, false
	}
	switch s[0] {
	case '"':
		if s[len(s)-1] != '"' {
			return s, false
		}
		return strings.ReplaceAll(s[1:len(s)-1], `\"`, `"`), true
	case '\'':
		if s[len(s)-1] != '\'' {
			return s, false
		}
		return s[1 : len(s)-1], true
	default:
		return s, false
	}
}

// runScheduler 用 robfig/cron 跑稳定的定时任务。
//
// 旧实现用 time.Sleep(wait) 在容器/虚机被挂起或系统时钟跳变时会丢任务；cron 库
// 内部按绝对时间和 timer 重新计算，并提供 SkipIfStillRunning + Recover 包装。
func runScheduler() {
	loadEnv()

	loc := schedulerLocation()
	spec := scheduleSpec()

	logger := cron.PrintfLogger(stdLogger())
	c := cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(),
		cron.WithLogger(logger),
		cron.WithChain(
			cron.Recover(logger),
			cron.SkipIfStillRunning(logger),
		),
	)

	id, err := c.AddFunc(spec, func() {
		fmt.Printf("[cron] 触发: %s\n", time.Now().In(loc).Format("2006-01-02 15:04:05"))
		if err := runOnce(false); err != nil {
			fmt.Fprintf(os.Stderr, "[cron] 任务失败: %v\n", err)
		}
	})
	if err != nil {
		fatal(fmt.Errorf("无效的 SCHEDULE_CRON=%q: %w", spec, err))
	}

	fmt.Printf("定时任务: %s (%s)\n", spec, loc.String())

	if envBool("RUN_ON_START", false) {
		fmt.Println("启动时立即执行一次...")
		if err := runOnce(false); err != nil {
			fmt.Fprintf(os.Stderr, "任务失败: %v\n", err)
		}
	}

	c.Start()
	if e := c.Entry(id); e.ID == id {
		fmt.Printf("下次执行: %s\n", e.Next.In(loc).Format("2006-01-02 15:04:05 MST"))
	}

	// runScheduler 在 server 模式由 goroutine 调起，cron 自带的 ticker 会阻塞
	// 自己的协程；外层 server 通过 ListenAndServe 阻塞。这里直接 select{} 防止
	// 单独 scheduler 模式时 main 退出。
	select {}
}

// scheduleSpec 返回 cron 表达式。优先 SCHEDULE_CRON；否则由
// SCHEDULE_HOUR/MINUTE/SECOND 推导成每日触发。
func scheduleSpec() string {
	if s := envRaw("SCHEDULE_CRON"); s != "" {
		return s
	}
	h := envInt("SCHEDULE_HOUR", 17)
	m := envInt("SCHEDULE_MINUTE", 0)
	s := envInt("SCHEDULE_SECOND", 0)
	return fmt.Sprintf("%d %d %d * * *", s, m, h)
}

// stdLogger 转发 cron 日志到 stdout，让容器日志一行不漏。
func stdLogger() cronStdlibLogger { return cronStdlibLogger{} }

type cronStdlibLogger struct{}

func (cronStdlibLogger) Printf(format string, args ...any) {
	fmt.Printf("[cron] "+format+"\n", args...)
}

func runOnce(dryRun bool) error {
	loadEnv()

	targetURL := strings.TrimSpace(os.Getenv("GRAFANA_URL"))
	if targetURL == "" {
		return fmt.Errorf("请在 .env 中配置 GRAFANA_URL（Dashboard 完整 URL）")
	}

	token := strings.TrimSpace(os.Getenv("GRAFANA_TOKEN"))
	if token == "" {
		return fmt.Errorf("请配置 GRAFANA_TOKEN（Grafana Service Account Token，glsa_ 开头）")
	}

	screenshotDir := envOr("SCREENSHOT_DIR", defaultScreenshotDir)
	if err := os.MkdirAll(screenshotDir, 0o755); err != nil {
		return err
	}

	ts := time.Now().Format("20060102-150405")
	outPath := filepath.Join(screenshotDir, fmt.Sprintf("grafana-%s.png", ts))

	fmt.Println("模式: agentless（Grafana /render API）")
	fmt.Println("截图:", targetURL)
	fmt.Printf("正在请求 Grafana 渲染（超时约 %d 秒，全页截图可能需 2～5 分钟）...\n", renderTimeoutSec()+30)
	if err := captureDashboard(targetURL, outPath, token); err != nil {
		return err
	}
	fmt.Println("已保存:", outPath)
	if n, err := pruneOldScreenshots(screenshotDir); err != nil {
		fmt.Fprintf(os.Stderr, "清理旧截图失败: %v\n", err)
	} else if n > 0 {
		fmt.Printf("已清理 %d 个超过保留期的截图\n", n)
	}

	if dryRun {
		fmt.Println("dry-run：跳过发送")
		return nil
	}

	fmt.Println("正在发送到 Lark...")
	if err := deliver(outPath, targetURL); err != nil {
		return err
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return envIntAlias(key, def)
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func envRaw(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// 兼容 grafana.env 中的 VIEWPORT_* / WAIT_MS 命名。
func envIntAlias(key string, def int) int {
	switch key {
	case "RENDER_WIDTH":
		if n := envInt("VIEWPORT_WIDTH", 0); n > 0 {
			return n
		}
	case "RENDER_TIMEOUT":
		if ms := envInt("WAIT_MS", 0); ms > 0 {
			sec := ms / 1000
			if sec < 30 {
				sec = 30
			}
			return sec
		}
	}
	return def
}

// renderHeight 默认 -1：Grafana 自动滚动截取整页 Dashboard。
func renderHeight() int {
	if !envBool("RENDER_FULL_PAGE", true) {
		if envRaw("RENDER_HEIGHT") != "" {
			return envInt("RENDER_HEIGHT", defaultRenderH)
		}
		if n := envInt("VIEWPORT_HEIGHT", 0); n > 0 {
			return n
		}
		return defaultRenderH
	}
	if envRaw("RENDER_HEIGHT") != "" {
		return envInt("RENDER_HEIGHT", -1)
	}
	return -1
}

func renderWidth() int {
	if envRaw("RENDER_WIDTH") != "" {
		return envInt("RENDER_WIDTH", defaultRenderW)
	}
	if n := envInt("VIEWPORT_WIDTH", 0); n > 0 {
		return n
	}
	return defaultRenderW
}

func renderTimeoutSec() int {
	if envRaw("RENDER_TIMEOUT") != "" {
		return envInt("RENDER_TIMEOUT", defaultRenderTimeout)
	}
	if ms := envInt("WAIT_MS", 0); ms > 0 {
		sec := ms / 1000
		if sec < 30 {
			sec = 30
		}
		if envBool("RENDER_FULL_PAGE", true) && sec < 120 {
			sec = 120
		}
		return sec
	}
	if envBool("RENDER_FULL_PAGE", true) {
		return defaultRenderTimeoutFullPage
	}
	return defaultRenderTimeout
}

func buildRenderURL(pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(u.Path, "/d/") {
		return "", fmt.Errorf("URL 不是 dashboard 路径 (/d/...): %s", pageURL)
	}
	u.Path = "/render" + u.Path
	q := u.Query()
	applyKioskQuery(q)
	q.Set("width", fmt.Sprintf("%d", renderWidth()))
	q.Set("height", fmt.Sprintf("%d", renderHeight()))
	q.Set("timeout", fmt.Sprintf("%d", renderTimeoutSec()))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// applyKioskQuery 添加 kiosk 参数，渲染时隐藏左侧菜单（通常也会隐藏顶栏）。
// RENDER_KIOSK：true（默认）、tv、false/0/no 关闭；URL 已含 kiosk 时不覆盖。
func applyKioskQuery(q url.Values) {
	if q.Has("kiosk") {
		return
	}
	switch strings.ToLower(strings.TrimSpace(envOr("RENDER_KIOSK", "true"))) {
	case "false", "0", "no", "off":
		return
	case "true", "1", "yes", "":
		q.Set("kiosk", "true")
	default:
		q.Set("kiosk", strings.TrimSpace(os.Getenv("RENDER_KIOSK")))
	}
}

func captureDashboard(pageURL, outPath, token string) error {
	renderURL, err := buildRenderURL(pageURL)
	if err != nil {
		return err
	}
	fmt.Println("渲染:", renderURL)

	timeoutSec := renderTimeoutSec()
	req, err := http.NewRequest(http.MethodGet, renderURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: time.Duration(timeoutSec+30) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("渲染请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("渲染 HTTP %d: %s", resp.StatusCode, truncate(string(body), 500))
	}
	if len(body) < 8 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
		return fmt.Errorf("渲染返回非 PNG（请确认 Grafana 已安装 image renderer）: %s", truncate(string(body), 300))
	}
	return os.WriteFile(outPath, body, 0o644)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func pruneOldScreenshots(dir string) (int, error) {
	days := envInt("SCREENSHOT_RETENTION_DAYS", defaultScreenshotRetentionDays)
	if days <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	var removed int
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasPrefix(ent.Name(), "grafana-") || !strings.HasSuffix(ent.Name(), ".png") {
			continue
		}
		info, err := ent.Info()
		if err != nil {
			return removed, err
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, ent.Name())); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func deliver(imagePath, grafanaURL string) error {
	appID, appSecret, chatID := larkConfig()
	if appID == "" || appSecret == "" || chatID == "" {
		return fmt.Errorf("请配置 LARK_APP_ID、LARK_APP_SECRET、LARK_CHAT_ID")
	}
	if err := sendLark(appID, appSecret, chatID, imagePath, grafanaURL); err != nil {
		return err
	}
	fmt.Println("已发送到 Lark 群")
	return nil
}

const defaultLarkAPIBase = "https://open.larksuite.com"

func larkAPIBase() string {
	return strings.TrimSuffix(strings.TrimSpace(envOr("LARK_API_BASE", defaultLarkAPIBase)), "/")
}

func larkConfig() (appID, appSecret, chatID string) {
	appID = strings.TrimSpace(envOr("LARK_APP_ID", envOr("LARK_APPID", "")))
	appSecret = strings.TrimSpace(envOr("LARK_APP_SECRET", envOr("LARK_APPSECRET", "")))
	chatID = strings.TrimSpace(os.Getenv("LARK_CHAT_ID"))
	return appID, appSecret, chatID
}

func sendLark(appID, appSecret, chatID, imagePath, grafanaURL string) error {
	token, err := larkTenantToken(appID, appSecret)
	if err != nil {
		return err
	}
	if err := larkSendText(token, chatID, larkCaption(grafanaURL)); err != nil {
		return err
	}
	imageKey, err := larkUploadImage(token, imagePath)
	if err != nil {
		return err
	}
	return larkSendImage(token, chatID, imageKey)
}

func larkCaption(grafanaURL string) string {
	title := envOr("LARK_MSG_TITLE", "最近24小时视频监控图表")
	return title + "\n时间 " + grafanaTimeRangeLabel(grafanaURL)
}

func grafanaTimeRangeLabel(grafanaURL string) string {
	loc := schedulerLocation()
	now := time.Now().In(loc)
	start, end := now.Add(-24*time.Hour), now

	u, err := url.Parse(grafanaURL)
	if err == nil {
		q := u.Query()
		if s, ok := parseGrafanaTimeExpr(q.Get("from"), now); ok {
			start = s
		}
		if e, ok := parseGrafanaTimeExpr(q.Get("to"), now); ok {
			end = e
		}
	}
	if end.Before(start) {
		start, end = end, start
	}
	const layout = "2006-01-02 15:04:05"
	return start.In(loc).Format(layout) + " - " + end.In(loc).Format(layout)
}

func schedulerLocation() *time.Location {
	loc := time.Local
	if tz := envRaw("TZ"); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			return l
		}
	}
	return loc
}

func parseGrafanaTimeExpr(expr string, now time.Time) (time.Time, bool) {
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "" || expr == "now" {
		return now, true
	}
	if !strings.HasPrefix(expr, "now-") {
		if ms, err := strconv.ParseInt(expr, 10, 64); err == nil {
			if ms > 1e12 {
				return time.UnixMilli(ms).In(now.Location()), true
			}
			return time.Unix(ms, 0).In(now.Location()), true
		}
		return time.Time{}, false
	}
	rest := strings.TrimPrefix(expr, "now-")
	for _, u := range []struct {
		suffix string
		d      time.Duration
	}{
		{"h", time.Hour},
		{"d", 24 * time.Hour},
		{"m", time.Minute},
	} {
		if strings.HasSuffix(rest, u.suffix) {
			n, err := strconv.Atoi(strings.TrimSuffix(rest, u.suffix))
			if err == nil && n >= 0 {
				return now.Add(-time.Duration(n) * u.d), true
			}
		}
	}
	return time.Time{}, false
}

func larkHTTPClient() *http.Client {
	sec := envInt("LARK_HTTP_TIMEOUT", 120)
	if sec <= 0 {
		sec = 120
	}
	return &http.Client{Timeout: time.Duration(sec) * time.Second}
}

func larkTenantToken(appID, appSecret string) (string, error) {
	base := larkAPIBase()
	body, _ := json.Marshal(map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	})
	req, err := http.NewRequest(http.MethodPost, base+"/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := larkHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Code != 0 {
		return "", fmt.Errorf("获取 token 失败 code=%d: %s", out.Code, out.Msg)
	}
	if out.TenantAccessToken == "" {
		return "", fmt.Errorf("获取 token 失败: %s", truncate(string(raw), 300))
	}
	return out.TenantAccessToken, nil
}

func larkUploadImage(token, imagePath string) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buf := &bytes.Buffer{}
	writer := multipart.NewWriter(buf)
	_ = writer.WriteField("image_type", "message")
	part, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, larkAPIBase()+"/open-apis/im/v1/images", buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := larkHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ImageKey string `json:"image_key"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if out.Code != 0 {
		return "", fmt.Errorf("上传图片失败 code=%d: %s", out.Code, out.Msg)
	}
	if out.Data.ImageKey == "" {
		return "", fmt.Errorf("上传图片失败: %s", truncate(string(raw), 300))
	}
	return out.Data.ImageKey, nil
}

func larkSendText(token, chatID, text string) error {
	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	return larkSendMessage(token, chatID, "text", string(content))
}

func larkSendImage(token, chatID, imageKey string) error {
	content, err := json.Marshal(map[string]string{"image_key": imageKey})
	if err != nil {
		return err
	}
	return larkSendMessage(token, chatID, "image", string(content))
}

func larkSendMessage(token, chatID, msgType, content string) error {
	payload, err := json.Marshal(map[string]string{
		"receive_id": chatID,
		"msg_type":   msgType,
		"content":    content,
	})
	if err != nil {
		return err
	}

	apiURL := larkAPIBase() + "/open-apis/im/v1/messages?receive_id_type=chat_id"
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := larkHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	if out.Code != 0 {
		return fmt.Errorf("发送 %s 失败 code=%d: %s", msgType, out.Code, out.Msg)
	}
	return nil
}
