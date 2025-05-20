package cargo_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/blocky/bkyc/internal/cargo"
	"github.com/blocky/bkyc/mocks"
)

type ClientThatErrors struct {
	errorMsg string
}

func (c ClientThatErrors) Do(_ *http.Request) (*http.Response, error) {
	return nil, errors.New(c.errorMsg)
}

func TestNewClientFromRaw(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// when
		got := cargo.NewClientFromRaw(http.DefaultClient, "sample-url", slog.Default())

		// then
		assert.NotEmpty(t, got)
	})
}

func TestNewClientWithLogger(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// when
		got := cargo.NewClientWithLogger(slog.Default())

		// then
		assert.NotEmpty(t, got)
	})
}

func TestClient_Compatible(t *testing.T) {
	clientVersion, err := strconv.ParseFloat(cargo.APIVersion, 64)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		minAPIVersion  float64
		maxAPIVersion  float64
		wantCompatible bool
	}{
		"happy path": {
			minAPIVersion:  clientVersion - 0.1,
			maxAPIVersion:  clientVersion + 0.1,
			wantCompatible: true,
		},
		"happy path - daemon too new": {
			minAPIVersion:  clientVersion + 0.1,
			maxAPIVersion:  clientVersion + 0.2,
			wantCompatible: false,
		},
		"happy path - daemon too old": {
			minAPIVersion:  clientVersion - 0.2,
			maxAPIVersion:  clientVersion - 0.1,
			wantCompatible: false,
		},
		"happy path - edge values": {
			minAPIVersion:  clientVersion,
			maxAPIVersion:  clientVersion,
			wantCompatible: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantPath := "/version"
			wantMethod := "GET"

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				_, err := fmt.Fprintf(
					w,
					`{"ApiVersion":"%f", "MinAPIVersion":"%f"}`,
					tc.maxAPIVersion,
					tc.minAPIVersion,
				)
				require.NoError(t, err)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			gotCompatible, err := sut.Compatible(context.Background())

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantCompatible, gotCompatible)
		})
	}

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"incorrect response": {
			srvResp:   `not-a-correct-response`,
			wantError: `decoding api version response`,
		},
		"incorrect min API version": {
			srvResp: fmt.Sprintf(
				`{"ApiVersion":"%s", "MinAPIVersion":"%s"}`,
				"1.5",
				"incorrect",
			),
			wantError: `invalid min api version`,
		},
		"incorrect max API version": {
			srvResp: fmt.Sprintf(
				`{"ApiVersion":"%s", "MinAPIVersion":"%s"}`,
				"incorect",
				"1.5",
			),
			wantError: `invalid max api version`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantPath := "/version"
			wantMethod := "GET"

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				_, err := fmt.Fprintln(w, tc.srvResp)
				require.NoError(t, err)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			_, gotErr := sut.Compatible(context.Background())

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "receiving api version",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding api version error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantPath := "/version"
			wantMethod := "GET"

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			_, gotErr := sut.Compatible(context.Background())

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		_, gotErr := sut.Compatible(context.Background())

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making api version request")
	})
}

func TestClient_ImageExistsByDigest(t *testing.T) {
	for name, tc := range map[string]struct {
		srvStatus int
		wantExist bool
	}{
		"happy path - image exists": {
			srvStatus: http.StatusOK,
			wantExist: true,
		},
		"happy path - image does not exist": {
			srvStatus: http.StatusNotFound,
			wantExist: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantDigest := "test-image-digest"
			wantPath := fmt.Sprintf("/v%s/images/%s/json", cargo.APIVersion, wantDigest)
			wantMethod := "GET"

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.srvStatus)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			gotExists, err := sut.ImageExists(context.Background(), wantDigest)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.wantExist, gotExists)
		})
	}

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "receiving image info",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding image info error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantMethod := "GET"
			wantDigest := "test-image-digest"
			wantPath := fmt.Sprintf("/v%s/images/%s/json", cargo.APIVersion, wantDigest)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			_, gotErr := sut.ImageExists(context.Background(), wantDigest)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		_, gotErr := sut.ImageExists(context.Background(), "digest")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making image info request")
	})
}

func TestClient_PullImage(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantImage := "test-image"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/images/create", cargo.APIVersion)
		wantPullStatus := []string{
			`Pulling from fake-library/not-real-image`,
			`Download complete`,
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantImage, r.URL.Query().Get("fromImage"))

			w.Header().Set("Content-Type", "application/json")

			for _, status := range wantPullStatus {
				_, err := fmt.Fprintf(w, `{"status": "%s"}`, status)
				require.NoError(t, err)
			}
		}))
		defer ts.Close()

		// expecting
		mockLogger := mocks.NewCargoLogger(t)
		for _, status := range wantPullStatus {
			mockLogger.EXPECT().
				Debug(status).
				Once()
		}

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, mockLogger)

		// when
		err := sut.PullImage(context.Background(), wantImage)

		// then
		require.NoError(t, err)
	})

	t.Run("error in pull stream", func(t *testing.T) {
		// given
		wantImage := "test-image"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/images/create", cargo.APIVersion)
		wantPullStatus := []string{
			`Pulling from fake-library/not-real-image`,
			`50% complete`,
		}
		wantPullError := "unexpected pull error in stream"

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantImage, r.URL.Query().Get("fromImage"))

			w.Header().Set("Content-Type", "application/json")

			for _, status := range wantPullStatus {
				_, err := fmt.Fprintf(w, `{"status": "%s"}`, status)
				require.NoError(t, err)
			}
			_, err := fmt.Fprintf(w, `{"error": "%s"}`, wantPullError)
			require.NoError(t, err)
		}))
		defer ts.Close()

		// expecting
		mockLogger := mocks.NewCargoLogger(t)
		for _, status := range wantPullStatus {
			mockLogger.EXPECT().
				Debug(status).
				Once()
		}

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, mockLogger)

		// when
		err := sut.PullImage(context.Background(), wantImage)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pulling image")
		assert.ErrorContains(t, err, wantPullError)
	})

	t.Run("error decoding pull stream", func(t *testing.T) {
		// given
		wantImage := "test-image"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/images/create", cargo.APIVersion)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantImage, r.URL.Query().Get("fromImage"))

			w.Header().Set("Content-Type", "application/json")

			_, err := fmt.Fprintln(w, `not-a-valid-json-response`)
			require.NoError(t, err)
		}))
		defer ts.Close()

		// expecting
		mockLogger := mocks.NewCargoLogger(t)
		mockLogger.AssertNotCalled(t, "Debug", mock.Anything)

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, mockLogger)

		// when
		err := sut.PullImage(context.Background(), wantImage)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "decoding image pull stream")
	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting image pull",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding image pull error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantImage := "test-image"
			wantMethod := "POST"
			wantPath := fmt.Sprintf("/v%s/images/create", cargo.APIVersion)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				assert.Equal(t, wantImage, r.URL.Query().Get("fromImage"))
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			gotErr := sut.PullImage(context.Background(), wantImage)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		gotErr := sut.PullImage(context.Background(), "image")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making image pull request")
	})
}

func sampleCfg() cargo.Config {
	return cargo.Config{
		Image:      "test-image",
		Name:       "test-name",
		User:       "test-user",
		Env:        []string{"test-env-key", "test-env-value"},
		Cmd:        []string{"test-cmd", "test-cmd-arg"},
		WorkingDir: "test-working-dir",
		Tmpfs:      map[string]string{"test-tempfs-key": "test-tempfs-value"},
		Binds: []string{
			"test-bind-path-host:/test-bind-path-guest",
		},
		AutoRemove: true,
	}
}

func TestNewCreateConfig(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		want := sampleCfg()

		// when
		got := cargo.NewCreateConfig(want)

		// then
		assert.Equal(t, want.Image, got.Image)
		assert.Equal(t, want.User, got.User)
		assert.Equal(t, want.Env, got.Env)
		assert.Equal(t, want.WorkingDir, got.WorkingDir)
		assert.Equal(t, want.Cmd, got.Cmd)

		assert.Equal(t, want.Tmpfs, got.HostConfig.Tmpfs)
		assert.Equal(t, want.Binds, got.HostConfig.Binds)
		assert.Equal(t, want.AutoRemove, got.HostConfig.AutoRemove)

		assert.True(t, got.Tty)

	})
}

func TestClient_Create(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantCfg := sampleCfg()
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/create", cargo.APIVersion)
		wantBody := map[string]any{
			"Image":      wantCfg.Image,
			"User":       wantCfg.User,
			"Env":        wantCfg.Env,
			"WorkingDir": wantCfg.WorkingDir,
			"Cmd":        wantCfg.Cmd,
			"HostConfig": map[string]any{
				"Tmpfs":      wantCfg.Tmpfs,
				"Binds":      wantCfg.Binds,
				"AutoRemove": wantCfg.AutoRemove,
			},
			"Tty": true,
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantCfg.Name, r.URL.Query().Get("name"))

			wantBodyJSON, err := json.Marshal(wantBody)
			assert.NoError(t, err)
			gotBodyJSON, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.JSONEq(t, string(wantBodyJSON), string(gotBodyJSON))

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_, err = fmt.Fprintf(w, `{"Id": "%s"}`, wantID)
			require.NoError(t, err)

		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		gotID, err := sut.Create(context.Background(), wantCfg)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantID, gotID)
	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting container create",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding container create error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantCfg := sampleCfg()
			wantMethod := "POST"
			wantPath := fmt.Sprintf("/v%s/containers/create", cargo.APIVersion)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				assert.Equal(t, wantCfg.Name, r.URL.Query().Get("name"))
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			_, gotErr := sut.Create(context.Background(), wantCfg)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error decoding response", func(t *testing.T) {
		// given
		wantCfg := sampleCfg()
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/create", cargo.APIVersion)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantCfg.Name, r.URL.Query().Get("name"))

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintln(w, `incorrect-create-response`)
			require.NoError(t, err)

		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		_, err := sut.Create(context.Background(), wantCfg)

		// then
		require.ErrorContains(t, err, "decoding container create body")
	})

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		_, gotErr := sut.Create(context.Background(), cargo.Config{})

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making container create request")
	})
}

func TestClient_Start(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/start", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		err := sut.Start(context.Background(), wantID)

		// then
		require.NoError(t, err)
	})

	t.Run("container already started", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/start", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			w.WriteHeader(http.StatusNotModified)
		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		err := sut.Start(context.Background(), wantID)

		// then
		require.ErrorContains(t, err, "container already started")
	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting container start",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding container start error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantID := "test-id"
			wantMethod := "POST"
			wantPath := fmt.Sprintf("/v%s/containers/%s/start", cargo.APIVersion, wantID)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			gotErr := sut.Start(context.Background(), wantID)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		gotErr := sut.Start(context.Background(), "id")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making container start request")
	})
}

func TestClient_Wait(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantStatusCode := 44
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/wait", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintf(
				w,
				`{"StatusCode": %d}`,
				wantStatusCode,
			)
			require.NoError(t, err)

		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		gotStatusCode, gotErrorMsg, err := sut.Wait(context.Background(), wantID)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantStatusCode, gotStatusCode)
		assert.Empty(t, gotErrorMsg)
	})

	t.Run("happy path - with error msg", func(t *testing.T) {
		// given
		wantStatusCode := 44
		wantErrorMsg := "test error info"
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/wait", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintf(
				w,
				`{"StatusCode": %d, "Error": {"Message": "%s"}}`,
				wantStatusCode,
				wantErrorMsg,
			)
			require.NoError(t, err)

		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		gotStatusCode, gotErrorMsg, err := sut.Wait(context.Background(), wantID)

		// then
		require.NoError(t, err)
		assert.Equal(t, wantStatusCode, gotStatusCode)
		assert.Equal(t, wantErrorMsg, gotErrorMsg)
	})

	t.Run("error decoding wait response", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/wait", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_, err := fmt.Fprintln(w, "not-a-correct-response")
			require.NoError(t, err)

		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		_, _, err := sut.Wait(context.Background(), wantID)

		// then
		require.Error(t, err)
		require.ErrorContains(t, err, "decoding wait response")

	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting container wait",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding container wait error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantID := "test-id"
			wantMethod := "POST"
			wantPath := fmt.Sprintf("/v%s/containers/%s/wait", cargo.APIVersion, wantID)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			_, _, gotErr := sut.Wait(context.Background(), wantID)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		_, _, gotErr := sut.Wait(context.Background(), "id")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making container wait request")
	})
}

func TestClient_GetLogs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "GET"
		wantStdOut := "true"
		wantStdErr := "true"
		wantPath := fmt.Sprintf("/v%s/containers/%s/logs", cargo.APIVersion, wantID)
		wantLogs := []string{"first line of logs", "second line of logs"}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantStdOut, r.URL.Query().Get("stdout"))
			assert.Equal(t, wantStdErr, r.URL.Query().Get("stderr"))

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			for _, logLine := range wantLogs {
				_, err := fmt.Fprintln(w, logLine)
				require.NoError(t, err)
			}

		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		gotLogs, err := sut.Logs(context.Background(), wantID)

		// then
		require.NoError(t, err)
		for _, wantLog := range wantLogs {
			assert.Contains(t, gotLogs, wantLog)
		}
	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting container logs",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding container logs error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantID := "test-id"
			wantMethod := "GET"
			wantStdOut := "true"
			wantStdErr := "true"
			wantPath := fmt.Sprintf("/v%s/containers/%s/logs", cargo.APIVersion, wantID)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				assert.Equal(t, wantStdOut, r.URL.Query().Get("stdout"))
				assert.Equal(t, wantStdErr, r.URL.Query().Get("stderr"))
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			_, gotErr := sut.Logs(context.Background(), wantID)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		_, gotErr := sut.Logs(context.Background(), "id")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making container logs request")
	})
}
func TestClient_Stop(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/stop", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		err := sut.Stop(context.Background(), wantID)

		// then
		require.NoError(t, err)
	})

	t.Run("container already stopped", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "POST"
		wantPath := fmt.Sprintf("/v%s/containers/%s/stop", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			w.WriteHeader(http.StatusNotModified)
		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		err := sut.Stop(context.Background(), wantID)

		// then
		require.ErrorContains(t, err, "container already stopped")
	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting container stop",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding container stop error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantID := "test-id"
			wantMethod := "POST"
			wantPath := fmt.Sprintf("/v%s/containers/%s/stop", cargo.APIVersion, wantID)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			gotErr := sut.Stop(context.Background(), wantID)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		gotErr := sut.Stop(context.Background(), "id")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making container stop request")
	})
}
func TestClient_Remove(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		wantID := "test-id"
		wantMethod := "DELETE"
		wantForceParam := "true"
		wantPath := fmt.Sprintf("/v%s/containers/%s", cargo.APIVersion, wantID)

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, wantMethod, r.Method)
			assert.Equal(t, wantPath, r.URL.Path)
			assert.Equal(t, wantForceParam, r.URL.Query().Get("force"))
			w.WriteHeader(http.StatusNoContent)
		}))
		defer ts.Close()

		sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

		// when
		err := sut.Remove(context.Background(), wantID)

		// then
		require.NoError(t, err)
	})

	for name, tc := range map[string]struct {
		srvResp   string
		wantError string
	}{
		"status not ok": {
			srvResp:   `{"message":"srv err msg"}`,
			wantError: "requesting container remove",
		},
		"incorrect srv error msg": {
			srvResp:   "incorrect srv error msg",
			wantError: "decoding container remove error",
		},
	} {
		t.Run(name, func(t *testing.T) {
			//given
			wantID := "test-id"
			wantMethod := "DELETE"
			wantForceParam := "true"
			wantPath := fmt.Sprintf("/v%s/containers/%s", cargo.APIVersion, wantID)

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, wantMethod, r.Method)
				assert.Equal(t, wantPath, r.URL.Path)
				assert.Equal(t, wantForceParam, r.URL.Query().Get("force"))
				http.Error(w, tc.srvResp, http.StatusTeapot)
			}))
			defer ts.Close()

			sut := cargo.NewClientFromRaw(ts.Client(), ts.URL, slog.Default())

			// when
			gotErr := sut.Remove(context.Background(), wantID)

			// then
			assert.ErrorContains(t, gotErr, tc.wantError)
		})
	}

	t.Run("error making request", func(t *testing.T) {
		// given
		wantError := "error making request"
		client := ClientThatErrors{
			errorMsg: wantError,
		}
		sut := cargo.NewClientFromRaw(client, "unused-url", slog.Default())

		// when
		gotErr := sut.Remove(context.Background(), "id")

		// then
		require.Error(t, gotErr)
		assert.ErrorContains(t, gotErr, wantError)
		assert.ErrorContains(t, gotErr, "making container remove request")
	})
}
