package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

const (
	PrivateKeyFile = "puff.key"
	PublicKeyFile  = "puff.pub"
)

// GenerateKeys creates a new Ed25519 key pair and saves them to disk
func GenerateKeys() error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	err = os.WriteFile(PrivateKeyFile, []byte(base64.StdEncoding.EncodeToString(priv)), 0600)
	if err != nil {
		return err
	}

	err = os.WriteFile(PublicKeyFile, []byte(base64.StdEncoding.EncodeToString(pub)), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Keys generated: %s, %s\n", PrivateKeyFile, PublicKeyFile)
	return nil
}

// LoadPrivateKey loads the private key from disk
func LoadPrivateKey() (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	return ed25519.PrivateKey(decoded), nil
}

// LoadPublicKey loads the public key from disk
func LoadPublicKey() (ed25519.PublicKey, error) {
	data, err := os.ReadFile(PublicKeyFile)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}
	return ed25519.PublicKey(decoded), nil
}

// Sign signs a message using the private key
func Sign(priv ed25519.PrivateKey, message string) string {
	sig := ed25519.Sign(priv, []byte(message))
	return base64.StdEncoding.EncodeToString(sig)
}

// Verify checks a signature against a message and public key
func Verify(pubKeyStr string, message string, sigStr string) bool {
	pubBytes, err := base64.StdEncoding.DecodeString(pubKeyStr)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sigStr)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(message), sigBytes)
}
