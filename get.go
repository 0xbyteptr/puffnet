package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func runGet() {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	server := getCmd.String("s", "wss://puff.vapma.wtf:8008/ws", "PuffNet WebSocket server")
	getCmd.Parse(os.Args[2:])

	if getCmd.NArg() < 1 {
		log.Fatal("Usage: puffctl get domain.tld [-s ws://host:port]")
	}
	domain := getCmd.Arg(0)

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		log.Fatal("WS dial error:", err)
	}
	// Ensure we send a close message before exiting to avoid "abnormal closure" on host
	defer func() {
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
		conn.Close()
	}()

	// Request info via fetch
	req := map[string]string{
		"type":   "fetch",
		"domain": domain,
	}
	data, _ := json.Marshal(req)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Fatal("WS write error:", err)
	}

	// Wait for response
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Fatal("WS read error:", err)
	}

	var resp ProxyResponse
	if err := json.Unmarshal(msg, &resp); err != nil {
		// If it's not a ProxyResponse, just print the raw message
		fmt.Println("----- Raw PuffNet Response -----")
		fmt.Println(string(msg))
		return
	}

	fmt.Println("----- PuffNet Response -----")
	fmt.Printf("Status: %d\n", resp.Status)
	fmt.Println(string(resp.Body))
}
