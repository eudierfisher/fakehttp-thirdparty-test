package example_test

import (
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/eudierfisher/fakehttp"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func gorillaWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to set websocket upgrade:", err)
		return
	}
	defer conn.Close()

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("server read error:", err)
			break
		}
		log.Println("server recv:", string(message))

		log.Println("server send:", string(message))
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			log.Println("server write error:", err)
			break
		}
	}
}

func TestGorillaWebsocket(t *testing.T) {
	// init HTTP server
	http.Handle("/ws", http.HandlerFunc(gorillaWebsocket))
	server := &http.Server{Addr: ":8080"}
	defer func() {
		t.Log("Shutting down server")
		server.Close()
	}()

	// init hub
	hub := fakehttp.NewHub()
	go func() {
		t.Log("Server is pretending to listen on 127.0.0.1:8080")
		server.Serve(hub.Listener())
	}()

	// replace dial func
	// either NetDial or NetDialContext is OK
	// use hub.Dial or hub.DialContext
	dialer := websocket.Dialer{NetDial: hub.Dial}
	conn, _, err := dialer.Dial("ws://127.0.0.1:8080/ws", nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	defer conn.Close()

	t.Log("Connected")
	go func() {
		for i := 0; i < 5; i++ {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					t.Log("Connection closed normally")
					return
				}
				t.Error("client read error:", err)
				return
			}
			t.Log("client recv:", string(message))
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for i := 0; i < 5; i++ {
		ts := time.Now().Format(time.Stamp)
		t.Log("client send:", ts)
		err = conn.WriteMessage(websocket.TextMessage, []byte(ts))
		if err != nil {
			t.Error("client write error:", err)
			t.FailNow()
		}
		<-ticker.C
	}
	t.Log("Closing connection")
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Error("write close:", err)
		t.FailNow()
	}
	conn.Close()
	t.Log("Done")
}
