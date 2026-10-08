package user_handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"star-fire/internal/models"
	"star-fire/pkg/public"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDirectBackendsHandler(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	server := &models.Server{DirectBackendDB: models.NewDirectBackendDB(db), TokenUsageDB: models.NewTokenUsageDB(db)}
	backend := &models.DirectBackend{ID: "backend", Name: "Platform", APIKey: "never-expose", BaseURL: "http://internal-host/v1", Enabled: true, Format: "openai", Models: []*public.Model{{Name: "model", IPPM: 2, OPPM: 6}}}
	if err := server.DirectBackendDB.Save(backend); err != nil {
		t.Fatal(err)
	}
	if err := server.LoadDirectBackends(); err != nil {
		t.Fatal(err)
	}
	if err := server.TokenUsageDB.SaveTokenUsage(&models.TokenUsage{ClientID: "direct:backend", Model: "model", TotalTokens: 12, Timestamp: time.Now()}); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/direct", NewMarketHandler(server).DirectBackendsHandler)
	for _, scenario := range []struct {
		query  string
		status int
		count  int
	}{
		{query: "", status: 400},
		{query: "?name=unknown", status: 200, count: 0},
		{query: "?name=model", status: 200, count: 1},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/direct"+scenario.query, nil))
		if recorder.Code != scenario.status {
			t.Fatalf("status=%d, body=%s", recorder.Code, recorder.Body.String())
		}
		if scenario.status == 200 {
			var response struct {
				Backends []models.DirectBackendView `json:"backends"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || len(response.Backends) != scenario.count {
				t.Fatalf("unexpected response: %s, %v", recorder.Body.String(), err)
			}
			if scenario.count > 0 && (response.Backends[0].Calls != 1 || response.Backends[0].TotalTokens != 12) {
				t.Fatalf("incorrect usage: %+v", response.Backends[0])
			}
		}
		if strings.Contains(recorder.Body.String(), "never-expose") || strings.Contains(recorder.Body.String(), "internal-host") {
			t.Fatal("direct endpoint exposed credentials or internal topology")
		}
	}
}
