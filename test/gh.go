package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	AsSupportedReleases = 3
	AsRepo              = "blocky/attestation-service-cli"
)

func PlatformDescription() string {
	return fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
}

type ReleaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type ReleaseInfo struct {
	Tag    string         `json:"tag_name"`
	Assets []ReleaseAsset `json:"assets"`
}

func (r *ReleaseInfo) Binary() *ReleaseAsset {
	for _, asset := range r.Assets {
		if strings.Contains(asset.Name, PlatformDescription()) {
			return &asset
		}
	}
	return nil
}

func (r *ReleaseInfo) Config() *ReleaseAsset {
	for _, asset := range r.Assets {
		if strings.Contains(asset.Name, "config.toml") {
			return &asset
		}
	}
	return nil
}

func GetAccessToken() string {
	return os.Getenv("GH_ACCESS_TOKEN")
}

func GetReleaseInfo(repo string, count int) ([]ReleaseInfo, error) {
	addr := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", repo, count)
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating api request: %w", err)
	}
	if GetAccessToken() != empty {
		req.Header.Set("Authorization", "token "+GetAccessToken())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting release info from '%s': %w", addr, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf(
				"reading release info response from '%s': %w",
				addr,
				err,
			)
		}
		return nil, fmt.Errorf(
			"getting release info from '%s', received status '%d', msg: '%s'",
			addr,
			resp.StatusCode,
			msg,
		)
	}
	var relInfo []ReleaseInfo
	err = json.NewDecoder(resp.Body).Decode(&relInfo)
	if err != nil {
		return nil, fmt.Errorf("decoding release info from '%s': %w", addr, err)
	}
	return relInfo, nil
}

func DownloadAsset(asset *ReleaseAsset, dstDir string) (string, error) {
	resp, err := http.Get(asset.URL)
	if err != nil {
		return "", fmt.Errorf("downloading asset '%s': %w", asset.URL, err)
	}
	defer resp.Body.Close()

	binPath := filepath.Join(dstDir, asset.Name)
	dstFile, err := os.Create(binPath)
	if err != nil {
		return "", fmt.Errorf("creating asset '%s': %w", asset.Name, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("copying asset '%s' to '%s': %w", asset.Name, binPath, err)
	}
	return binPath, nil
}

type Release struct {
	tag        string
	binPath    string
	configPath string
}

func DownloadReleases(infos []ReleaseInfo, dstPath string) ([]Release, error) {
	var downloaded []Release
	for _, releaseInfo := range infos {
		releaseDir := filepath.Join(dstPath, releaseInfo.Tag)
		err := os.MkdirAll(releaseDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("creating release dir: %w", err)
		}

		binary := releaseInfo.Binary()
		binFile, err := DownloadAsset(binary, releaseDir)
		if err != nil {
			return nil, fmt.Errorf("downloading binary: %w", err)
		}

		config := releaseInfo.Config()
		configFile, err := DownloadAsset(config, releaseDir)
		if err != nil {
			return nil, fmt.Errorf("downloading config: %w", err)
		}

		downloaded = append(
			downloaded,
			Release{
				tag:        releaseInfo.Tag,
				binPath:    binFile,
				configPath: configFile,
			},
		)
	}
	return downloaded, nil
}
