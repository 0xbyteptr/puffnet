package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func runGet() {
	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	server := getCmd.String("s", "wss://puff.vapma.wtf:443/ws", "PuffNet WebSocket server")
	getCmd.Parse(os.Args[2:])

	if getCmd.NArg() < 1 {
		log.Fatal("Usage: puffctl get domain.tld [-s ws://host:port]")
	}
	domainArg := getCmd.Arg(0)
	domain := domainArg
	path := "/"
	if i := strings.Index(domainArg, "/"); i != -1 {
		domain = domainArg[:i]
		path = domainArg[i:]
	}

	conn, dialResp, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		if dialResp != nil {
			log.Fatalf("WS dial error: %v (Status: %s)", err, dialResp.Status)
		}
		log.Fatal("WS dial error:", err)
	}
	// Ensure we send a close message before exiting to avoid "abnormal closure" on host
	defer func() {
		msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(time.Second))
		conn.Close()
	}()

	// Request info via fetch
	req := WSMessage{
		Type:   "fetch",
		Domain: domain,
		Path:   path,
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
