package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/winkor4/taktaev_project_sp56/internal/pkg/config"
	"github.com/winkor4/taktaev_project_sp56/internal/server"
	"github.com/winkor4/taktaev_project_sp56/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {

	srv := newTestSrv(t)
	testAuth(t, srv)

}

func newTestSrv(t *testing.T) *httptest.Server {

	cfg, err := config.Parse()
	if err != nil {
		require.NoError(t, err)
	}

	require.NotEmpty(t, cfg.DatabaseURI)

	db, err := storage.New(cfg.DatabaseURI)
	if err != nil {
		require.NoError(t, err)
	}

	err = db.Truncate()
	require.NoError(t, err)

	srv := server.New(server.Config{
		Cfg: cfg,
		DB:  db,
	})

	return httptest.NewServer(server.SrvRouter(srv))
}

func testAuth(t *testing.T, srv *httptest.Server) {

	type (
		want struct {
			statusCode int
		}

		testData struct {
			name string
			path string
			body []byte
			want want
		}

		reqSchema struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
	)

	reqData := make(map[int][]byte)

	var logPass []byte
	logPass, err := json.Marshal(reqSchema{
		Login:    "ivan",
		Password: "1234",
	})
	require.NoError(t, err)
	reqData[0] = logPass

	logPass, err = json.Marshal(reqSchema{
		Login:    "ii",
		Password: "",
	})
	require.NoError(t, err)
	reqData[1] = logPass

	logPass, err = json.Marshal(reqSchema{
		Login:    "ivan",
		Password: "3215",
	})
	require.NoError(t, err)
	reqData[2] = logPass

	testTable := []testData{
		{
			name: "POST /api/user/register",
			path: "/api/user/register",
			body: reqData[0],
			want: want{statusCode: http.StatusOK},
		},
		{
			name: "повторный POST /api/user/register",
			path: "/api/user/register",
			body: reqData[0],
			want: want{statusCode: http.StatusConflict},
		},
		{
			name: "bad req POST /api/user/register",
			path: "/api/user/register",
			body: reqData[1],
			want: want{statusCode: http.StatusBadRequest},
		},
		{
			name: "POST /api/user/login",
			path: "/api/user/login",
			body: reqData[2],
			want: want{statusCode: http.StatusUnauthorized},
		},
		{
			name: "POST /api/user/login",
			path: "/api/user/login",
			body: reqData[0],
			want: want{statusCode: http.StatusOK},
		},
	}

	for _, testData := range testTable {
		t.Run(testData.name, func(t *testing.T) {

			body := bytes.NewReader(testData.body)
			request, err := http.NewRequest(http.MethodPost, srv.URL+testData.path, body)
			require.NoError(t, err)
			request.Header.Set("Content-Type", "application/json")

			client := srv.Client()
			r, err := client.Do(request)
			require.NoError(t, err)

			assert.Equal(t, testData.want.statusCode, r.StatusCode)

			err = r.Body.Close()
			require.NoError(t, err)

		})
	}

}
