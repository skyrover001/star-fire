package models

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	configs "star-fire/config"
	"star-fire/pkg/public"

	"github.com/gorilla/websocket"
)

func TestSendControlSerializesWrites(t *testing.T) {
	previousEnabled := configs.Config.ControlWriterEnabled
	previousBufferSize := configs.Config.ControlWriterBufSize
	configs.Config.ControlWriterEnabled = true
	configs.Config.ControlWriterBufSize = 256
	t.Cleanup(func() {
		configs.Config.ControlWriterEnabled = previousEnabled
		configs.Config.ControlWriterBufSize = previousBufferSize
	})

	const messages = 128
	received := make(chan struct{}, messages)
	upgrader := websocket.Upgrader{}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			var message public.WSMessage
			if err := conn.ReadJSON(&message); err != nil {
				return
			}
			received <- struct{}{}
		}
	}))
	defer httpServer.Close()

	wsURL := "ws" + httpServer.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	client := NewClient("writer-test", "127.0.0.1", conn)
	t.Cleanup(func() {
		client.ControlConnMutex.Lock()
		client.ControlConn = nil
		client.ControlConnMutex.Unlock()
		client.StopControlWriter()
		_ = conn.Close()
	})

	var senders sync.WaitGroup
	for i := 0; i < messages; i++ {
		senders.Add(1)
		go func() {
			defer senders.Done()
			if err := client.SendControl(public.WSMessage{Type: public.KEEPALIVE}); err != nil {
				t.Errorf("SendControl: %v", err)
			}
		}()
	}
	senders.Wait()
	for i := 0; i < messages; i++ {
		select {
		case <-received:
		case <-time.After(2 * time.Second):
			t.Fatalf("received %d of %d messages", i, messages)
		}
	}
}

func TestStopControlWriterMakesSendUnavailable(t *testing.T) {
	previousEnabled := configs.Config.ControlWriterEnabled
	configs.Config.ControlWriterEnabled = true
	t.Cleanup(func() { configs.Config.ControlWriterEnabled = previousEnabled })

	client := &Client{}
	client.StartControlWriter()
	client.StopControlWriter()
	if err := client.SendControl(public.WSMessage{Type: public.KEEPALIVE}); err == nil {
		t.Fatal("SendControl after StopControlWriter succeeded")
	}
}
