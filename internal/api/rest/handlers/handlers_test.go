package handlers

import (
	"context"
	"encoding/json"
	"github.com/danilovkiri/dk_go_url_shortener/internal/api/rest/model"
	"github.com/danilovkiri/dk_go_url_shortener/internal/config"
	shortenerService "github.com/danilovkiri/dk_go_url_shortener/internal/service/shortener"
	"github.com/danilovkiri/dk_go_url_shortener/internal/service/shortener/v1"
	"github.com/danilovkiri/dk_go_url_shortener/internal/storage"
	"github.com/danilovkiri/dk_go_url_shortener/internal/storage/infile"
	"github.com/go-chi/chi"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type HandlersTestSuite struct {
	suite.Suite
	storage          storage.URLStorage
	shortenerService shortenerService.Processor
	urlHandler       *URLHandler
	router           *chi.Mux
	ts               *httptest.Server
}

func (suite *HandlersTestSuite) SetupTest() {
	cfg, _ := config.NewDefaultConfiguration()
	// parsing flags causes flag redefined errors
	//cfg.ParseFlags()
	suite.storage, _ = infile.InitStorage(cfg.StorageConfig)
	suite.shortenerService, _ = shortener.InitShortener(suite.storage)
	suite.urlHandler, _ = InitURLHandler(suite.shortenerService, cfg.ServerConfig)
	suite.router = chi.NewRouter()
	suite.ts = httptest.NewServer(suite.router)
}

// TestHandlersTestSuite initializes test suite for being accessible
func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

func (suite *HandlersTestSuite) TestHandleGetURL() {
	ctx := context.Background()
	sURL, _ := suite.shortenerService.Encode(ctx, "https://yandex.ru")
	suite.router.Get("/{urlID}", suite.urlHandler.HandleGetURL())

	// set tests' parameters
	type want struct {
		code int
	}
	tests := []struct {
		name string
		sURL string
		want want
	}{
		{
			name: "Correct GET query",
			sURL: sURL,
			want: want{
				code: 307,
			},
		},
		{
			name: "Invalid GET query",
			sURL: "",
			want: want{
				code: 404,
			},
		},
	}

	// perform each test
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			client := resty.New()
			client.SetRedirectPolicy(resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}))
			res, err := client.R().SetPathParams(map[string]string{"urlID": tt.sURL}).Get(suite.ts.URL + "/{urlID}")
			if err != nil {
				t.Fatalf(err.Error())
			}
			assert.Equal(t, tt.want.code, res.StatusCode())
		})
	}
	defer suite.ts.Close()
}

func (suite *HandlersTestSuite) TestHandlePostURL() {
	suite.router.Post("/", suite.urlHandler.HandlePostURL())

	// set tests' parameters
	type want struct {
		code int
	}
	tests := []struct {
		name string
		URL  string
		want want
	}{
		{
			name: "Correct POST query",
			URL:  "https://www.yandex.ru",
			want: want{
				code: 201,
			},
		},
		{
			name: "Invalid POST query (empty query)",
			URL:  "",
			want: want{
				code: 400,
			},
		},
		{
			name: "Invalid POST query (not URL)",
			URL:  "kke738enb734b",
			want: want{
				code: 400,
			},
		},
	}

	// perform each test
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			payload := strings.NewReader(tt.URL)
			client := resty.New()
			res, err := client.R().SetBody(payload).Post(suite.ts.URL)
			if err != nil {
				t.Fatalf("Could not create POST request")
			}
			assert.Equal(t, tt.want.code, res.StatusCode())
		})
	}
	defer suite.ts.Close()
}

func (suite *HandlersTestSuite) TestJSONHandlePostURL() {
	suite.router.Post("/api/shorten", suite.urlHandler.JSONHandlePostURL())

	// set tests' parameters
	type want struct {
		code int
	}
	tests := []struct {
		name string
		URL  model.RequestURL
		want want
	}{
		{
			name: "Correct POST query",
			URL: model.RequestURL{
				URL: "https://www.yandex.ru",
			},
			want: want{
				code: 201,
			},
		},
		{
			name: "Invalid POST query (empty query)",
			URL: model.RequestURL{
				URL: "",
			},
			want: want{
				code: 400,
			},
		},
		{
			name: "Invalid POST query (not URL)",
			URL: model.RequestURL{
				URL: "kke738enb734b",
			},
			want: want{
				code: 400,
			},
		},
	}

	// perform each test
	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(tt.URL)
			payload := strings.NewReader(string(reqBody))
			client := resty.New()
			res, err := client.R().SetBody(payload).Post(suite.ts.URL + "/api/shorten")
			if err != nil {
				t.Fatalf("Could not perform JSON POST request")
			}
			t.Logf(string(res.Body()))
			assert.Equal(t, tt.want.code, res.StatusCode())
		})
	}
	defer suite.ts.Close()
}
