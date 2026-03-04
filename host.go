package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const OwnershipFile = "ownership.json"

// ProxyRequest is the message sent from Host to Serve client
type ProxyRequest struct {
	ID      string              `json:"id"`
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

// ProxyResponse is the message sent from Serve client to Host
type ProxyResponse struct {
	ID      string              `json:"id,omitempty"`
	Status  int                 `json:"status"`
	Headers map[string][]string `json:"headers"`
	Body    []byte              `json:"body"`
}

// Tunnel represents a connected 'serve' client
type Tunnel struct {
	Conn      *websocket.Conn
	Domain    string
	Mu        sync.Mutex
	Responses map[string]chan *ProxyResponse
}

type WSMessage struct {
	Type      string `json:"type"`
	Domain    string `json:"domain"`
	PublicKey string `json:"public_key,omitempty"`
	Signature string `json:"signature,omitempty"`
}

var (
	tunnels         = make(map[string]*Tunnel)
	tunnelsMu       sync.RWMutex
	domainOwnership = make(map[string]string) // domain -> publicKey (base64)
	ownershipMu     sync.RWMutex
	upgrader        = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func loadOwnership() {
	ownershipMu.Lock()
	defer ownershipMu.Unlock()

	data, err := os.ReadFile(OwnershipFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("Error reading ownership file: %v", err)
		return
	}

	if err := json.Unmarshal(data, &domainOwnership); err != nil {
		log.Printf("Error unmarshaling ownership data: %v", err)
	}
}

func saveOwnership() {
	ownershipMu.RLock()
	defer ownershipMu.RUnlock()

	data, err := json.MarshalIndent(domainOwnership, "", "  ")
	if err != nil {
		log.Printf("Error marshaling ownership data: %v", err)
		return
	}

	if err := os.WriteFile(OwnershipFile, data, 0644); err != nil {
		log.Printf("Error writing ownership file: %v", err)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}
	defer conn.Close()

	var currentTunnel *Tunnel

	for {
		messageType, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WS read error: %v", err)
			}
			if currentTunnel != nil {
				tunnelsMu.Lock()
				if tunnels[currentTunnel.Domain] == currentTunnel {
					delete(tunnels, currentTunnel.Domain)
				}
				tunnelsMu.Unlock()
			}
			return
		}

		if messageType == websocket.BinaryMessage {
			var resp ProxyResponse
			if err := json.Unmarshal(rawMsg, &resp); err == nil && resp.ID != "" {
				if currentTunnel != nil {
					currentTunnel.Mu.Lock()
					ch, ok := currentTunnel.Responses[resp.ID]
					if ok {
						ch <- &resp
						delete(currentTunnel.Responses, resp.ID)
					}
					currentTunnel.Mu.Unlock()
				}
			}
			continue
		}

		var msg WSMessage
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "register":
			if msg.Domain == "" || msg.PublicKey == "" || msg.Signature == "" {
				log.Printf("Registration failed: missing domain, key, or signature")
				conn.Close()
				return
			}

			// Verify if the requester actually owns the private key for this domain
			if !Verify(msg.PublicKey, msg.Domain, msg.Signature) {
				log.Printf("Registration failed: invalid signature for domain %s", msg.Domain)
				conn.Close()
				return
			}

			ownershipMu.Lock()
			existingKey, exists := domainOwnership[msg.Domain]
			saveNeeded := false
			if exists {
				if existingKey != msg.PublicKey {
					ownershipMu.Unlock()
					log.Printf("Registration denied: domain %s is already owned by another key", msg.Domain)
					conn.Close()
					return
				}
			} else {
				domainOwnership[msg.Domain] = msg.PublicKey
				log.Printf("Domain %s first-time registration, ownership locked to provided key", msg.Domain)
				saveNeeded = true
			}
			ownershipMu.Unlock()

			if saveNeeded {
				saveOwnership()
			}

			currentTunnel = &Tunnel{
				Conn:      conn,
				Domain:    msg.Domain,
				Responses: make(map[string]chan *ProxyResponse),
			}
			tunnelsMu.Lock()
			tunnels[msg.Domain] = currentTunnel
			tunnelsMu.Unlock()
			log.Printf("Tunnel registered and verified: %s\n", msg.Domain)

		case "fetch":
			handleFetch(conn, msg.Domain)
		}
	}
}

func handleFetch(conn *websocket.Conn, domain string) {
	tunnelsMu.RLock()
	tunnel, ok := tunnels[domain]
	tunnelsMu.RUnlock()

	if !ok {
		resp := ProxyResponse{Status: 404, Body: []byte("No tunnel for " + domain)}
		out, _ := json.Marshal(resp)
		conn.WriteMessage(websocket.BinaryMessage, out)
		return
	}

	reqID := uuid.New().String()
	respCh := make(chan *ProxyResponse, 1)

	tunnel.Mu.Lock()
	tunnel.Responses[reqID] = respCh
	tunnel.Mu.Unlock()

	req := ProxyRequest{
		ID:     reqID,
		Method: "GET",
		Path:   "/",
	}

	data, _ := json.Marshal(req)
	tunnel.Mu.Lock()
	err := tunnel.Conn.WriteMessage(websocket.BinaryMessage, data)
	tunnel.Mu.Unlock()

	if err != nil {
		resp := ProxyResponse{Status: 502, Body: []byte("Tunnel write error")}
		out, _ := json.Marshal(resp)
		conn.WriteMessage(websocket.BinaryMessage, out)
		return
	}

	resp := <-respCh
	out, _ := json.Marshal(resp)
	conn.WriteMessage(websocket.BinaryMessage, out)
	log.Printf("Proxying fetch for %s complete\n", domain)
}

func runHost() {
	loadOwnership()

	port := "8008"
	if len(os.Args) >= 3 {
		port = os.Args[2]
	}

	http.HandleFunc("/ws", wsHandler)

	log.Println("PuffNet Host on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
