package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dropserve/internal/control"
)

const (
	defaultBaseURL     = "http://127.0.0.1:8080"
	defaultPublicPort  = 8080
	defaultOpenMinutes = 15
)

func RunOpen(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("open", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var reusable bool
	var minutes int
	var portOverride int
	fs.IntVar(&minutes, "minutes", defaultOpenMinutes, "Minutes to keep portal open")
	fs.IntVar(&minutes, "m", defaultOpenMinutes, "Alias for --minutes")
	fs.BoolVar(&reusable, "reusable", false, "Allow multiple claims")
	fs.BoolVar(&reusable, "reuseable", false, "Alias for --reusable")
	fs.BoolVar(&reusable, "r", false, "Alias for --reusable")
	policy := fs.String("policy", "overwrite", "Default conflict policy: overwrite or autorename")
	hostOverride := fs.String("host", "", "Override LAN host/IP for printed link")
	fs.IntVar(&portOverride, "port", 0, "Override server port for control call + printed link")

	if err := fs.Parse(args); err != nil {
		return err
	}

	policyValue := strings.ToLower(strings.TrimSpace(*policy))
	if policyValue != "overwrite" && policyValue != "autorename" {
		return fmt.Errorf("policy must be overwrite or autorename")
	}

	destAbs, err := canonicalizeCwd()
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	baseURL := resolveBaseURL(portOverride)

	request := control.CreatePortalRequest{
		DestAbs:              destAbs,
		OpenMinutes:          minutes,
		Reusable:             reusable,
		DefaultPolicy:        policyValue,
		AutorenameOnConflict: policyValue == "autorename",
	}

	response, err := createPortal(baseURL, request)
	if err != nil {
		return err
	}

	host, err := resolveHost(*hostOverride)
	if err != nil {
		fmt.Fprintf(stderr, "warning: %v; falling back to 127.0.0.1\n", err)
		host = "127.0.0.1"
	}

	port := publicPortFromEnv(portOverride)
	link := formatPortalURL(host, port, response.PortalID)
	fmt.Fprintln(stdout, link)
	if host != "localhost" {
		localLink := formatPortalURL("localhost", port, response.PortalID)
		fmt.Fprintln(stdout, localLink)
	}
	return nil
}

func canonicalizeCwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}

	return resolved, nil
}

func resolveBaseURL(portOverride int) string {
	if portOverride > 0 {
		return fmt.Sprintf("http://127.0.0.1:%d", portOverride)
	}
	raw := strings.TrimSpace(os.Getenv("DROPSERVE_URL"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("DROPSERVE_ADDR"))
	}
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("DROPSERVE_PUBLIC_ADDR"))
	}
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("DROPSERVE_CONTROL_URL"))
	}
	return normalizeBaseURL(raw)
}

func normalizeBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultBaseURL
	}

	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		if parsed, err := url.Parse(trimmed); err == nil {
			if parsed.Hostname() == "0.0.0.0" {
				parsed.Host = "127.0.0.1" + portSuffix(parsed.Port())
				return strings.TrimRight(parsed.String(), "/")
			}
		}
		return strings.TrimRight(trimmed, "/")
	}

	trimmed = strings.TrimRight(trimmed, "/")
	if host, port, err := net.SplitHostPort(trimmed); err == nil {
		if host == "0.0.0.0" {
			return "http://127.0.0.1:" + port
		}
		return "http://" + trimmed
	}

	if trimmed == "0.0.0.0" {
		return "http://127.0.0.1"
	}

	return "http://" + trimmed
}

func portSuffix(port string) string {
	if port == "" {
		return ""
	}
	return ":" + port
}

func createPortal(baseURL string, payload control.CreatePortalRequest) (control.CreatePortalResponse, error) {
	endpoint := strings.TrimRight(baseURL, "/") + "/api/control/portals"
	body, err := json.Marshal(payload)
	if err != nil {
		return control.CreatePortalResponse{}, fmt.Errorf("encode request: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return control.CreatePortalResponse{}, fmt.Errorf("build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return control.CreatePortalResponse{}, fmt.Errorf("control api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		message := strings.TrimSpace(string(bodyBytes))
		if message == "" {
			message = resp.Status
		}
		return control.CreatePortalResponse{}, fmt.Errorf("control api error: %s", message)
	}

	var response control.CreatePortalResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return control.CreatePortalResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return response, nil
}

func resolveHost(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override), nil
	}

	ip, err := DetectPrimaryIPv4()
	if err != nil {
		return "", err
	}

	return ip.String(), nil
}

func publicPortFromEnv(portOverride int) int {
	if portOverride > 0 {
		return portOverride
	}
	if port := strings.TrimSpace(os.Getenv("DROPSERVE_PORT")); port != "" {
		if portValue, err := strconv.Atoi(port); err == nil {
			return portValue
		}
	}

	addr := strings.TrimSpace(os.Getenv("DROPSERVE_ADDR"))
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("DROPSERVE_PUBLIC_ADDR"))
	}
	if addr == "" {
		return defaultPublicPort
	}

	if strings.Contains(addr, "://") {
		if parsed, err := url.Parse(addr); err == nil {
			if parsed.Port() == "" {
				return defaultPublicPort
			}
			if portValue, err := strconv.Atoi(parsed.Port()); err == nil {
				return portValue
			}
			return defaultPublicPort
		}
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return defaultPublicPort
	}

	portValue, err := strconv.Atoi(port)
	if err != nil {
		return defaultPublicPort
	}

	return portValue
}

func formatPortalURL(host string, port int, portalID string) string {
	if port == 80 {
		return fmt.Sprintf("http://%s/p/%s", host, portalID)
	}

	return fmt.Sprintf("http://%s:%d/p/%s", host, port, portalID)
}
