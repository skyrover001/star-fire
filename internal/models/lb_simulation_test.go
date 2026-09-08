package models

import (
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
)

const (
	simulationDirectBackends = 50
	simulationContributors   = 1000
	simulationRequests       = 10000
)

type simulationMetrics struct {
	mu            sync.Mutex
	direct        int
	community     int
	unavailable   int
	directHits    map[string]int
	communityHits map[string]int
}

type simulationCapacityMetrics struct {
	overloadedResources int
	overCapacity        int
	maxOverCapacity     int
}

func (m *simulationMetrics) recordDirect(backend *DirectBackend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.direct++
	m.directHits[backend.ID]++
}

func (m *simulationMetrics) recordCommunity(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.community++
	m.communityHits[client.ID]++
}

func (m *simulationMetrics) recordUnavailable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unavailable++
}

func TestLoadBalanceSimulation(t *testing.T) {
	if os.Getenv("RUN_LB_SIM") != "1" {
		t.Skip("set RUN_LB_SIM=1 to run the 10,000-request routing simulation")
	}

	originalConfig := configs.Config
	defer func() { configs.Config = originalConfig }()
	configs.Config.LBScoreOffline = true
	configs.Config.LBCooldownEnabled = false
	configs.Config.LBJitter = 0.05
	configs.Config.LBCandidateCount = 3
	configs.Config.LBBalancedMinScore = 0.5
	configs.Config.LBHRWSubsetSize = 0

	previousLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(previousLogOutput)

	server, online, offline := newSimulationServer(t)
	requests := simulationRequestsForBurst()
	t.Logf("topology: direct=%d, contributors=%d (online=%d offline=%d), requests/scenario=%d", simulationDirectBackends, simulationContributors, online, offline, simulationRequests)

	for _, routing := range []string{"stability", "cost", "balanced"} {
		resetSimulationLoad(server)
		metrics := runSimulationBurst(server, routing, requests)
		if got := metrics.direct + metrics.community + metrics.unavailable; got != simulationRequests {
			t.Fatalf("%s: accounted requests = %d, want %d", routing, got, simulationRequests)
		}
		capacity := simulationOverCapacity(server)
		t.Logf("%s: direct=%d (%.2f%%), community=%d (%.2f%%), unavailable=%d, direct-hotspots=%s, community-hotspots=%s, overloaded-resources=%d, over-capacity=%d, max-over-capacity=%d",
			routing,
			metrics.direct, percent(metrics.direct, simulationRequests),
			metrics.community, percent(metrics.community, simulationRequests),
			metrics.unavailable,
			topSimulationHits(metrics.directHits),
			topSimulationHits(metrics.communityHits),
			capacity.overloadedResources,
			capacity.overCapacity,
			capacity.maxOverCapacity,
		)
	}
}

func newSimulationServer(t *testing.T) (*Server, int, int) {
	t.Helper()
	server := &Server{LoadBalanceAlgorithm: "smart"}
	server.clients.Store(map[string]map[string]*Client{})
	server.directBackends = make(map[string][]*DirectBackend)

	rng := rand.New(rand.NewSource(20260402))
	modelNames := simulationModelNames()
	for index := 0; index < simulationDirectBackends; index++ {
		backend := &DirectBackend{
			ID:       fmt.Sprintf("direct-%02d", index),
			Name:     fmt.Sprintf("Direct %02d", index),
			Enabled:  true,
			MaxConns: 80,
			Priority: 1 + index%3,
			Models:   simulationDirectModels(index, modelNames),
		}
		backend.SetHealthy(true)
		backend.SetLatencyEMA(30 + float64(index%8)*25)
		for sample := 0; sample < 12; sample++ {
			if index%10 == 0 {
				backend.UpdateReliability(0)
			} else {
				backend.UpdateReliability(1)
			}
		}
		for _, model := range backend.Models {
			server.directBackends[model.Name] = append(server.directBackends[model.Name], backend)
		}
	}

	clientBuckets := make(map[string]map[string]*Client, len(modelNames))
	for _, model := range modelNames {
		clientBuckets[model] = make(map[string]*Client)
	}
	online, offline := 0, 0
	for index := 0; index < simulationContributors; index++ {
		// Current availability is sampled from a normal-distribution churn snapshot.
		onlineNow := rng.NormFloat64() < 0.90
		status := "offline"
		if onlineNow {
			status = "online"
			online++
		} else {
			offline++
		}
		client := &Client{
			ID:            fmt.Sprintf("community-%04d", index),
			Status:        status,
			ControlConn:   &websocket.Conn{},
			Latency:       boundedInt(250+int(rng.NormFloat64()*180), 20, 5000),
			BandwidthMbps: boundedFloat(45+rng.NormFloat64()*20, 5, 120),
			Models:        simulationCommunityModels(rng, index, modelNames),
		}
		client.SetCachedScores(
			boundedFloat(0.78+rng.NormFloat64()*0.15, 0.05, 1),
			boundedFloat(0.72+rng.NormFloat64()*0.18, 0.05, 1),
		)
		for _, model := range client.Models {
			clientBuckets[model.Name][client.ID] = client
		}
	}
	server.clients.Store(clientBuckets)
	return server, online, offline
}

func simulationModelNames() []string {
	return []string{
		"qwen3-8b", "qwen3-14b", "qwen3-32b", "llama-3.3-70b", "deepseek-r1", "glm-4.5", "mistral-small",
		"gemma-3-27b", "phi-4", "command-r", "yi-lightning", "minimax-m2",
	}
}

func simulationDirectModels(index int, modelNames []string) []*public.Model {
	models := make([]*public.Model, 0, 8)
	for modelIndex := 0; modelIndex < 7; modelIndex++ {
		models = append(models, simulationModel(modelNames[modelIndex], modelIndex, index%3))
	}
	models = append(models, simulationModel(modelNames[7+index%5], 7+index%5, index%3))
	return models
}

func simulationCommunityModels(rng *rand.Rand, index int, modelNames []string) []*public.Model {
	want := 4 + index%3
	chosen := map[int]bool{}
	for len(chosen) < want {
		chosen[rng.Intn(len(modelNames))] = true
	}
	models := make([]*public.Model, 0, want)
	for modelIndex := range chosen {
		models = append(models, simulationModel(modelNames[modelIndex], modelIndex, (index+modelIndex)%3))
	}
	return models
}

func simulationModel(name string, modelIndex, priceTier int) *public.Model {
	base := 1.4 + float64(modelIndex)*0.20 + float64(priceTier)*0.25
	return &public.Model{Name: name, IPPM: base, OPPM: base * 2.2, CIPPM: base * 0.4}
}

func simulationRequestsForBurst() []string {
	modelNames := simulationModelNames()
	rng := rand.New(rand.NewSource(20260403))
	requests := make([]string, simulationRequests)
	for index := range requests {
		if rng.Intn(100) < 78 {
			requests[index] = modelNames[rng.Intn(7)]
		} else {
			requests[index] = modelNames[7+rng.Intn(5)]
		}
	}
	return requests
}

func runSimulationBurst(server *Server, routing string, requests []string) *simulationMetrics {
	metrics := &simulationMetrics{directHits: map[string]int{}, communityHits: map[string]int{}}
	start := make(chan struct{})
	var group sync.WaitGroup
	for requestIndex, model := range requests {
		group.Add(1)
		go func(requestIndex int, model string) {
			defer group.Done()
			<-start
			userID := fmt.Sprintf("consumer-%05d", requestIndex)
			var client *Client
			var backend *DirectBackend
			switch routing {
			case "stability":
				backend = server.PickDirect(model, userID, nil)
				if backend == nil {
					client = server.LoadBalanceWithTolerance(model, userID, nil, 0, 0)
				}
			case "cost":
				client, backend = server.PickCheapest(model, userID, nil)
			case "balanced":
				client = server.LoadBalanceBalanced(model, userID, nil, 0, 0)
				if client == nil {
					backend = server.PickDirect(model, userID, nil)
				}
			}
			if backend != nil {
				backend.IncrActive()
				metrics.recordDirect(backend)
				return
			}
			if client != nil {
				client.IncrActiveConnections()
				metrics.recordCommunity(client)
				return
			}
			metrics.recordUnavailable()
		}(requestIndex, model)
	}
	close(start)
	group.Wait()
	return metrics
}

func resetSimulationLoad(server *Server) {
	server.directBackendsMu.RLock()
	backends := map[string]*DirectBackend{}
	for _, list := range server.directBackends {
		for _, backend := range list {
			backends[backend.ID] = backend
		}
	}
	server.directBackendsMu.RUnlock()
	for _, backend := range backends {
		for backend.GetActive() > 0 {
			backend.DecrActive()
		}
	}
	for _, clients := range server.clients.Load().(map[string]map[string]*Client) {
		for _, client := range clients {
			for client.GetActiveConnections() > 0 {
				client.DecrActiveConnections()
			}
		}
	}
}

func simulationOverCapacity(server *Server) simulationCapacityMetrics {
	metrics := simulationCapacityMetrics{}
	seenClients := map[string]*Client{}
	for _, clients := range server.clients.Load().(map[string]map[string]*Client) {
		for _, client := range clients {
			seenClients[client.ID] = client
		}
	}
	for _, client := range seenClients {
		over := int(client.GetActiveConnections()) - server.effectiveMaxConnections(client)
		if over > 0 {
			metrics.overloadedResources++
			metrics.overCapacity += over
			metrics.maxOverCapacity = max(metrics.maxOverCapacity, over)
		}
	}
	seenBackends := map[string]*DirectBackend{}
	server.directBackendsMu.RLock()
	for _, backends := range server.directBackends {
		for _, backend := range backends {
			seenBackends[backend.ID] = backend
		}
	}
	server.directBackendsMu.RUnlock()
	for _, backend := range seenBackends {
		limit := backend.MaxConns
		if limit <= 0 {
			limit = 1
		}
		over := int(backend.GetActive()) - limit
		if over > 0 {
			metrics.overloadedResources++
			metrics.overCapacity += over
			metrics.maxOverCapacity = max(metrics.maxOverCapacity, over)
		}
	}
	return metrics
}

func topSimulationHits(hits map[string]int) string {
	type hit struct {
		id    string
		count int
	}
	all := make([]hit, 0, len(hits))
	for id, count := range hits {
		all = append(all, hit{id: id, count: count})
	}
	sort.Slice(all, func(left, right int) bool {
		if all[left].count == all[right].count {
			return all[left].id < all[right].id
		}
		return all[left].count > all[right].count
	})
	if len(all) > 3 {
		all = all[:3]
	}
	return fmt.Sprintf("%v", all)
}

func boundedInt(value, min, max int) int {
	return int(math.Max(float64(min), math.Min(float64(max), float64(value))))
}

func boundedFloat(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}

func percent(value, total int) float64 {
	return float64(value) * 100 / float64(total)
}
