package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

func runPost() {
	postCmd := flag.NewFlagSet("post", flag.ExitOnError)
	server := postCmd.String("s", "ws://ssh.byteptr.xyz:8008/ws", "PuffNet WS server")
	body := postCmd.String("d", "", "POST body")
	postCmd.Parse(os.Args[2:])

	if postCmd.NArg() < 1 {
		log.Fatal("Usage: puffctl post domain.tld -d 'body' [-s ws://host:port]")
	}
	domainArg := postCmd.Arg(0)
	domain := domainArg
	path := "/"
	if i := strings.Index(domainArg, "/"); i != -1 {
		domain = domainArg[:i]
		path = domainArg[i:]
	}

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		log.Fatal("WS dial error:", err)
	}
	defer conn.Close()

	req := WSMessage{
		Type:   "post",
		Domain: domain,
		Path:   path,
		Body:   *body,
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
	if err := json.Unmarshal(msg, &resp); err != nil {
		fmt.Println("----- Raw PuffNet Response -----")
		fmt.Println(string(msg))
		return
	}

	fmt.Println("----- PuffNet POST Response -----")
	fmt.Printf("Status: %d\n", resp.Status)
	fmt.Println(string(resp.Body))
}
