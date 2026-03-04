package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gorilla/websocket"
)

func runPost() {
	postCmd := flag.NewFlagSet("post", flag.ExitOnError)
	server := postCmd.String("s", "wss://puff.vapma.wtf:8008/ws", "PuffNet WS server")
	body := postCmd.String("d", "", "POST body")
	postCmd.Parse(os.Args[2:])

	if postCmd.NArg() < 1 {
		log.Fatal("Usage: puffctl post domain.tld -d 'body' [-s ws://host:port]")
	}
	domain := postCmd.Arg(0)

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		log.Fatal("WS dial error:", err)
	}
	defer conn.Close()

	req := map[string]string{
		"type":   "post",
		"domain": domain,
		"body":   *body,
	}
	data, _ := json.Marshal(req)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Fatal("WS write error:", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Fatal("WS read error:", err)
	}

	var resp ProxyResponse
	json.Unmarshal(msg, &resp)

	fmt.Println("----- PuffNet POST Response -----")
	fmt.Printf("Status: %d\n%s\n", resp.Status, resp.Body)
}
