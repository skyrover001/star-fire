package models

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"star-fire/pkg/public"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

func TestUpdateModelPricePushesUpdateToConnectedClient(t *testing.T) {
	upgrader := websocket.Upgrader{}
	serverConnCh := make(chan *websocket.Conn, 1)
	httpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(writer, request, nil)
		if err != nil {
			t.Errorf("upgrade websocket: %v", err)
			return
		}
		serverConnCh <- conn
	}))
	defer httpServer.Close()

	reader, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(httpServer.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer reader.Close()
	serverConn := <-serverConnCh
	defer serverConn.Close()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	clientDB := NewClientDB(db)
	connectedClient := &Client{
		ID: "client-1", UserID: "user-1", Status: "online", ControlConn: serverConn,
		Models: []*public.Model{{Name: "model-a", Engine: "ollama", IPPM: 1}},
	}
	server := &Server{ClientDB: clientDB}
	server.clients.Store(map[string]map[string]*Client{
		"model-a": {"client-1": connectedClient},
	})

	updated, err := server.UpdateModelPrice("user-1", "model-a", 4.2, 8.4, 1.2)
	if err != nil {
		t.Fatalf("update model price: %v", err)
	}
	if updated != 1 {
		t.Fatalf("updated clients: got %d, want 1", updated)
	}

	if err := reader.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var message public.WSMessage
	if err := reader.ReadJSON(&message); err != nil {
		t.Fatalf("read price update: %v", err)
	}
	if message.Type != public.MODEL_PRICE_UPDATE {
		t.Fatalf("message type: got %q, want %q", message.Type, public.MODEL_PRICE_UPDATE)
	}
	content := message.Content.(map[string]interface{})
	if content["model"] != "model-a" || content["ippm"] != 4.2 || content["oppm"] != 8.4 || content["cippm"] != 1.2 {
		t.Fatalf("unexpected price update content: %#v", content)
	}
}

func TestRegisterModelReplacesReconnectedClientInstance(t *testing.T) {
	server := &Server{}
	server.clients.Store(map[string]map[string]*Client{})
	oldClient := &Client{ID: "same-client", Models: []*public.Model{{Name: "model-a", OPPM: 8.3}}}
	newClient := &Client{ID: "same-client", Models: []*public.Model{{Name: "model-a", OPPM: 4}}}
	model := &public.Model{Name: "model-a"}

	server.RegisterModel(model, oldClient)
	server.RegisterModel(model, newClient)

	if got := server.GetClientByModel("model-a", "same-client"); got != newClient {
		t.Fatalf("registered client instance = %p, want replacement %p", got, newClient)
	}
	server.RemoveClientInstance("model-a", oldClient)
	if got := server.GetClientByModel("model-a", "same-client"); got != newClient {
		t.Fatalf("old connection cleanup removed replacement: got %p, want %p", got, newClient)
	}
}
