package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUsageSupplySeparation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	usageDB := NewTokenUsageDB(db)
	now := time.Now()
	for _, clientID := range []string{"personal-client", "direct:backend", ""} {
		if err := usageDB.SaveTokenUsage(&TokenUsage{RequestID: clientID, UserID: "consumer", ClientID: clientID, Model: "model", InputTokens: 10, OutputTokens: 2, TotalTokens: 12, IPPM: 2, OPPM: 6, Cost: 0.000032, Timestamp: now}); err != nil {
			t.Fatal(err)
		}
	}
	start, end := now.Add(-time.Hour), now.Add(time.Hour)
	market, err := usageDB.GetModelMarketStats(start, end)
	if err != nil || len(market) != 1 {
		t.Fatalf("market stats: %+v, %v", market, err)
	}
	if market[0].ClientCount != 1 {
		t.Fatalf("personal client count = %d, want 1", market[0].ClientCount)
	}
	if market[0].DirectCount != 1 {
		t.Fatalf("direct count = %d, want 1", market[0].DirectCount)
	}
	for _, getStats := range []func() (map[string]float64, error){
		func() (map[string]float64, error) { return usageDB.GetUsageTotalStatsByUserID("consumer") },
		func() (map[string]float64, error) { return usageDB.GetUsageStatsByUserID("consumer", start, end) },
	} {
		stats, err := getStats()
		if err != nil || stats["client_count"] != 1 || stats["direct_count"] != 1 || stats["total_calls"] != 3 {
			t.Fatalf("incorrect supply counts: %+v, %v", stats, err)
		}
	}
	direct, err := usageDB.GetDirectBackendUsage("model", start, end)
	if err != nil || len(direct) != 1 || direct["backend"].Calls != 1 || direct["backend"].TotalTokens != 12 {
		t.Fatalf("incorrect direct usage: %+v, %v", direct, err)
	}
	income, err := usageDB.GetIncomeStatsByTimeRange([]string{"personal-client", "direct:backend", ""}, start, end)
	if err != nil || income["total_calls"] != 1 {
		t.Fatalf("personal income must exclude direct and unknown sources: %+v, %v", income, err)
	}
	clientDB := NewClientDB(db)
	for _, clientID := range []string{"personal-client", "direct:backend"} {
		if err := clientDB.SaveClient(&Client{ID: clientID, UserID: "owner"}); err != nil {
			t.Fatal(err)
		}
	}
	totals, err := usageDB.GetTotalIncomeStatsByUserID("owner", clientDB)
	if err != nil || totals["total_calls"] != 1 || totals["client_count"] != 1 || totals["unique_users"] != 1 || totals["max_income"] != 0.000032 || totals["min_income"] != 0.000032 {
		t.Fatalf("incorrect personal lifetime income stats: %+v, %v", totals, err)
	}
	models, err := usageDB.GetUsageStatsByModel("consumer", start, end)
	if err != nil || len(models) != 1 || models[0].ClientCount != 1 || models[0].DirectCount != 1 || models[0].UncachedInputCost <= 0 || models[0].OutputCost <= 0 {
		t.Fatalf("incorrect usage model breakdown: %+v, %v", models, err)
	}
	usage, total, err := usageDB.GetUserTokenUsagePaged("consumer", start, end, 1, 10)
	if err != nil || len(usage) != 3 || total != 3 {
		t.Fatalf("consumer usage must retain all sources: %+v, total=%d, err=%v", usage, total, err)
	}
}
