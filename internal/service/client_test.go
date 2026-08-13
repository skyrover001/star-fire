package service

import (
	"strconv"
	"testing"
	"time"

	"star-fire/internal/models"
	"star-fire/pkg/public"
)

func TestHeartbeatLatencyIgnoresUnsolicitedModelUpdate(t *testing.T) {
	client := &models.Client{}
	pong := &public.PPMessage{Timestamp: strconv.FormatInt(time.Now().UnixMilli(), 10)}

	if latency, matched := heartbeatLatency(client, pong); matched || latency != 0 {
		t.Fatalf("unsolicited model update matched heartbeat: latency=%d matched=%v", latency, matched)
	}
}

func TestHeartbeatLatencyMatchesCurrentPing(t *testing.T) {
	client := &models.Client{LastPingTime: time.Now().Add(-10 * time.Millisecond).UnixMilli()}
	pong := &public.PPMessage{Timestamp: strconv.FormatInt(client.LastPingTime, 10)}

	latency, matched := heartbeatLatency(client, pong)
	if !matched || latency < 0 || latency > 1000 {
		t.Fatalf("current heartbeat not matched: latency=%d matched=%v", latency, matched)
	}
}
