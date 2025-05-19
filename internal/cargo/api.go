package cargo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

const APIVersion = "1.47"

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
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient HTTPDoer
	BaseURL    string
	log        Logger
}

func NewClientFromRaw(
	httpClient HTTPDoer,
	baseURL string,
	log Logger,
) *Client {
	return &Client{
		httpClient: httpClient,
		BaseURL:    baseURL,
		log:        log,
	}
}

func newUnixSockHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
}

func NewClientWithLogger(log Logger) *Client {
	return NewClientFromRaw(
		newUnixSockHTTPClient(),
		"http://cargo.unix.sock",
		log,
	)
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	body any,
) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling api request body: %v", err)
		}
		reader = bytes.NewReader(b)
	}

	baseURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing api base URL: %v", err)
	}

	pathURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing api path URL: %v", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		baseURL.ResolveReference(pathURL).String(),
		reader,
	)
	if err != nil {
		return nil, fmt.Errorf("making api request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

type ServerMsg struct {
	Msg string `json:"message"`
}

func (c *Client) Compatible(ctx context.Context) (bool, error) {
	resp, err := c.do(
		ctx,
		"GET",
		"/version",
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("making api version request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return false, fmt.Errorf("decoding api version error: %w", err)
		}
		return false, fmt.Errorf("receiving api version: %s", info.Msg)
	}

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

	if minVer <= currVer && currVer <= maxVer {
		return true, nil
	}

	return false, nil
}

func (c *Client) ImageExists(
	ctx context.Context,
	digest string,
) (bool, error) {
	resp, err := c.do(
		ctx,
		"GET",
		fmt.Sprintf("/v%s/images/%s/json", APIVersion, digest),
		nil,
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
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return false, fmt.Errorf("decoding image info error: %w", err)
		}
		return false, fmt.Errorf("receiving image info: %s", info.Msg)
	}
}

func (c *Client) PullImage(ctx context.Context, image string) error {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf(
			"/v%s/images/create?fromImage=%s",
			APIVersion,
			url.QueryEscape(image),
		),
		nil,
	)
	if err != nil {
		return fmt.Errorf("making image pull request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return fmt.Errorf("decoding image pull error: %w", err)
		}
		return fmt.Errorf("requesting image pull: %s", info.Msg)
	}

	decoder := json.NewDecoder(resp.Body)
	for decoder.More() {
		var msg map[string]any
		if err := decoder.Decode(&msg); err != nil {
			return fmt.Errorf("decoding image pull stream: %w", err)
		}
		if errStr, ok := msg["error"]; ok {
			return fmt.Errorf("pulling image: %v", errStr)
		}
		c.log.Debug(fmt.Sprintf("%s", msg["status"]))
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
	Tty        bool       `json:"Tty"`
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
		Tty: true,
	}
}

func (c *Client) Create(
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
	)
	if err != nil {
		return "", fmt.Errorf("making container create request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return "", fmt.Errorf("decoding container create error: %w", err)
		}
		return "", fmt.Errorf("requesting container create: %s", info.Msg)
	}

	var res struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decoding container create body: %w", err)
	}
	return res.ID, nil
}

func (c *Client) Start(ctx context.Context, cID string) error {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf("/v%s/containers/%s/start", APIVersion, cID),
		nil,
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
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return fmt.Errorf("decoding container start error: %w", err)
		}
		return fmt.Errorf("requesting container start: %s", info.Msg)
	}
}

func (c *Client) Wait(ctx context.Context, cID string) (int, string, error) {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf("/v%s/containers/%s/wait", APIVersion, cID),
		nil,
	)
	if err != nil {
		return 0, "", fmt.Errorf("making container wait request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return 0, "", fmt.Errorf("decoding container wait error: %w", err)
		}
		return 0, "", fmt.Errorf("requesting container wait: %s", info.Msg)
	}

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

func (c *Client) Logs(
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
	)
	if err != nil {
		return "", fmt.Errorf("making container logs request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return "", fmt.Errorf("decoding container logs error: %w", err)
		}
		return "", fmt.Errorf("requesting container logs: %s", info.Msg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading container logs body: %w", err)
	}
	return string(body), nil
}

func (c *Client) Stop(ctx context.Context, cID string) error {
	resp, err := c.do(
		ctx,
		"POST",
		fmt.Sprintf("/v%s/containers/%s/stop?t=5", APIVersion, cID),
		nil,
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
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return fmt.Errorf("decoding container stop error: %w", err)
		}
		return fmt.Errorf("requesting container stop: %s", info.Msg)
	}
}

func (c *Client) Remove(ctx context.Context, cID string) error {
	resp, err := c.do(
		ctx,
		"DELETE",
		fmt.Sprintf("/v%s/containers/%s?force=true", APIVersion, cID),
		nil,
	)
	if err != nil {
		return fmt.Errorf("making container remove request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		var info ServerMsg
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return fmt.Errorf("decoding container remove error: %w", err)
		}
		return fmt.Errorf("requesting container remove: %s", info.Msg)
	}
	return nil
}
