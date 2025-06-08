package container

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
)

const APIVersion = "1.47"
const empty = ""

type Config struct {
	Image      string
	Name       string
	User       string
	Env        []string
	Cmd        []string
	WorkingDir string
	Tmpfs      map[string]string
	Binds      []string
	AutoRemove bool
}

type Logger interface {
	Debug(string, ...any)
	Warn(string, ...any)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type APIClient struct {
	Doer    HTTPDoer
	BaseURL string
	Log     Logger
}

func NewAPIClient() *APIClient {
	return &APIClient{
		Doer:    newUnixSockHTTPClient(),
		BaseURL: "http://placeholder.for.unix.sock",
		Log:     slog.Default(),
	}
}

func NewAPIClientWithLogger(log Logger) *APIClient {
	return &APIClient{
		Doer:    newUnixSockHTTPClient(),
		BaseURL: "http://placeholder.for.unix.sock",
		Log:     log,
	}
}

func getDaemonSocketPath() string {
	defaultSocket := "/var/run/docker.sock"
	if host := os.Getenv("DOCKER_HOST"); host != empty {
		dh, err := url.Parse(host)
		if err != nil || dh.Scheme != "unix" {
			return defaultSocket
		}
		return dh.Path
	}
	return defaultSocket
}

func newUnixSockHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", getDaemonSocketPath())
			},
		},
	}
}

type ServerMsg struct {
	Msg string `json:"message"`
}

func (c *APIClient) do(
	ctx context.Context,
	method string,
	path string,
	body any,
	fwdStatuses ...int,
) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling api request body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	baseURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing api base URL: %w", err)
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing api path URL: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		baseURL.ResolveReference(pathURL).String(),
		reader,
	)
	if err != nil {
		return nil, fmt.Errorf("preparing api request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making api request: %w", err)
	}

	if len(fwdStatuses) > 0 && !slices.Contains(fwdStatuses, resp.StatusCode) {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return nil, fmt.Errorf("decoding api response error: %w", err)
		}
		return nil, fmt.Errorf("unknown response: %s", info.Msg)
	}

	return resp, nil
}

func (c *APIClient) Compatible(ctx context.Context) (bool, error) {
	resp, err := c.do(
		ctx,
		"GET",
		"/version",
		nil,
		http.StatusOK,
	)
	if err != nil {
		return false, fmt.Errorf("making api version request: %w", err)
	}
	defer resp.Body.Close()

	var v struct {
		APIVersion    string `json:"ApiVersion"`
		MinAPIVersion string `json:"MinAPIVersion"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return false, fmt.Errorf("decoding api version response: %w", err)
	}

	minVer, err := strconv.ParseFloat(v.MinAPIVersion, 32)
	if err != nil {
		return false, fmt.Errorf("invalid min api version: %w", err)
	}

	maxVer, err := strconv.ParseFloat(v.APIVersion, 32)
	if err != nil {
		return false, fmt.Errorf("invalid max api version: %w", err)
	}

	currVer, err := strconv.ParseFloat(APIVersion, 32)
	if err != nil {
		return false, fmt.Errorf("invalid client api version: %w", err)
	}

	if currVer < minVer || maxVer < currVer {
		return false, nil
	}

	return true, nil
}

func (c *APIClient) ImageExists(
	ctx context.Context,
	digest string,
) (bool, error) {
	resp, err := c.do(
		ctx,
		"GET",
		fmt.Sprintf("/v%s/images/%s/json", APIVersion, digest),
		nil,
		http.StatusOK,
		http.StatusNotFound,
	)
	if err != nil {
		return false, fmt.Errorf("making image info request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf(
			"unknown image info response code: %d",
			resp.StatusCode,
		)
	}
}

func (c *APIClient) PullImage(ctx context.Context, image string) error {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf(
			"/v%s/images/create?fromImage=%s",
			APIVersion,
			url.QueryEscape(image),
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return fmt.Errorf("making image pull request: %w", err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg map[string]any
		if err := decoder.Decode(&msg); err != nil {
			return fmt.Errorf("decoding image pull stream: %w", err)
		}
		if errStr, ok := msg["error"]; ok {
			return fmt.Errorf("pulling image: %s", errStr)
		}
		c.Log.Debug(fmt.Sprintf("%s", msg["status"]))
	}
	return nil
}

type HostConfig struct {
	Tmpfs      map[string]string `json:"Tmpfs"`
	Binds      []string          `json:"Binds"`
	AutoRemove bool              `json:"AutoRemove"`
}

type CreateConfig struct {
	Image      string     `json:"Image"`
	User       string     `json:"User"`
	Env        []string   `json:"Env"`
	WorkingDir string     `json:"WorkingDir"`
	Cmd        []string   `json:"Cmd"`
	HostConfig HostConfig `json:"HostConfig"`
	TTY        bool       `json:"Tty"`
}

func NewCreateConfig(in Config) CreateConfig {
	return CreateConfig{
		Image:      in.Image,
		User:       in.User,
		Env:        in.Env,
		WorkingDir: in.WorkingDir,
		Cmd:        in.Cmd,
		HostConfig: HostConfig{
			Tmpfs:      in.Tmpfs,
			Binds:      in.Binds,
			AutoRemove: in.AutoRemove,
		},
		TTY: true,
	}
}

func (c *APIClient) Create(
	ctx context.Context,
	cfg Config,
) (string, error) {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf(
			"/v%s/containers/create?name=%s",
			APIVersion,
			url.QueryEscape(cfg.Name),
		),
		NewCreateConfig(cfg),
		http.StatusCreated,
	)
	if err != nil {
		return "", fmt.Errorf("making container create request: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decoding container create body: %w", err)
	}
	return res.ID, nil
}

func (c *APIClient) Start(ctx context.Context, cID string) error {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf("/v%s/containers/%s/start", APIVersion, cID),
		nil,
		http.StatusNoContent,
		http.StatusNotModified,
	)
	if err != nil {
		return fmt.Errorf("making container start request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotModified:
		return fmt.Errorf("container already started '%s'", cID)
	default:
		return fmt.Errorf("unknown container start response code: %d", resp.StatusCode)
	}
}

func (c *APIClient) Wait(ctx context.Context, cID string) (int, string, error) {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf("/v%s/containers/%s/wait", APIVersion, cID),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return 0, "", fmt.Errorf("making container wait request: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		StatusCode int `json:"StatusCode"`
		Error      struct {
			Message string `json:"Message"`
		} `json:"Error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, "", fmt.Errorf("decoding wait response: %w", err)
	}
	return res.StatusCode, res.Error.Message, nil
}

func (c *APIClient) Logs(
	ctx context.Context,
	cID string,
) (string, error) {
	resp, err := c.do(
		ctx,
		"GET",
		fmt.Sprintf(
			"/v%s/containers/%s/logs?stdout=true&stderr=true",
			APIVersion,
			cID,
		),
		nil,
		http.StatusOK,
	)
	if err != nil {
		return "", fmt.Errorf("making container logs request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading container logs body: %w", err)
	}
	return string(body), nil
}

func (c *APIClient) Stop(ctx context.Context, cID string) error {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf("/v%s/containers/%s/stop?t=5", APIVersion, cID),
		nil,
		http.StatusNoContent,
		http.StatusNotModified,
	)
	if err != nil {
		return fmt.Errorf("making container stop request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotModified:
		return fmt.Errorf("container already stopped '%s'", cID)
	default:
		return fmt.Errorf("unknown container stop response code: %d", resp.StatusCode)
	}
}

func (c *APIClient) Remove(ctx context.Context, cID string) error {
	resp, err := c.do(
		ctx,
		"DELETE",
		fmt.Sprintf("/v%s/containers/%s?force=true", APIVersion, cID),
		nil,
		http.StatusNoContent,
	)
	if err != nil {
		return fmt.Errorf("making container remove request: %w", err)
	}
	defer resp.Body.Close()

	return nil
}
