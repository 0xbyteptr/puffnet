package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

func runServe() {
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	domain := serveCmd.String("d", "", "public domain")
	target := serveCmd.String("h", "localhost:3000", "local host:port")
	server := serveCmd.String("s", "wss://puff.vapma.wtf:8008/ws", "puff host websocket")
	serveCmd.Parse(os.Args[2:])

	if *domain == "" {
		log.Fatal("Usage: puffctl serve -d domain.tld -h host:port [-s wss://host:port]")
	}

	// Load keys for signing registration
	privKey, err := LoadPrivateKey()
	if err != nil {
		log.Fatalf("Error loading private key: %v. Please run 'keygen' first.", err)
	}
	signature := Sign(privKey, *domain)
	pubKeyBase64, err := os.ReadFile(PublicKeyFile)
	if err != nil {
		log.Fatalf("Error reading public key file: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		log.Fatal("WS dial error:", err)
	}
	defer conn.Close()

	var mu sync.Mutex

	// Register tunnel with host using signature to prove ownership
	reg := WSMessage{
		Type:      "register",
		Domain:    *domain,
		PublicKey: string(pubKeyBase64),
		Signature: signature,
	}
	regData, _ := json.Marshal(reg)
	mu.Lock()
	conn.WriteMessage(websocket.TextMessage, regData)
	mu.Unlock()

	log.Printf("Serving http://%s as %s via %s (Securely Registered)\n", *target, *domain, *server)

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WS connection closed:", err)
			break
		}

		// Host sends ProxyRequest via BinaryMessage
		if messageType != websocket.BinaryMessage {
			continue
		}

		var req ProxyRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			log.Println("Unmarshal ProxyRequest error:", err)
			continue
		}

		// Handle request in a goroutine for concurrency
		go func(request ProxyRequest) {
			url := "http://" + *target + request.Path
			httpReq, err := http.NewRequest(request.Method, url, bytes.NewReader(request.Body))
			if err != nil {
				log.Println("Internal request creation error:", err)
				return
			}

			// Copy headers from proxy request to local request
			for k, v := range request.Headers {
				for _, val := range v {
					httpReq.Header.Add(k, val)
				}
			}

			resp, err := http.DefaultClient.Do(httpReq)
			var response ProxyResponse
			if err != nil {
				response = ProxyResponse{
					ID:     request.ID,
					Status: 502,
					Body:   []byte("Bad Gateway: " + err.Error()),
				}
			} else {
				bodyResp, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				response = ProxyResponse{
					ID:      request.ID,
					Status:  resp.StatusCode,
					Headers: resp.Header,
					Body:    bodyResp,
				}
			}

			out, _ := json.Marshal(response)
			mu.Lock()
			if err := conn.WriteMessage(websocket.BinaryMessage, out); err != nil {
				log.Println("WS write response error:", err)
			}
			mu.Unlock()
		}(req)
	}
}
