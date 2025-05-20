package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var WebSocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := WebSocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error while upgrading to websocket:", err)
		return nil, err
	}
	return conn, nil
}

// Close websocket connection
func CloseConnection(conn *websocket.Conn) {
	if conn != nil {
		err := conn.Close()
		if err != nil {
			log.Println("Error while closing websocket connection:", err)
		}
	}
}
