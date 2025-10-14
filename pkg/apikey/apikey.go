package apikey

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/nrmnqdds/gomaluum/internal/errors"
)

const (
	// DefaultAPIKey is used when no x-gomaluum-key header is provided
	DefaultAPIKey = "gomaluum-default-key-2024"
	APIKeyLength  = 32 // 32 bytes = 256 bits
)

// GenerateAPIKey generates a new random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, APIKeyLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", errors.ErrFailedToGenerateAPIKey
	}

	// Convert to hex string for easier handling
	return hex.EncodeToString(bytes), nil
}

// GenerateTimestampedAPIKey generates an API key with timestamp prefix for uniqueness
func GenerateTimestampedAPIKey() (string, error) {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, APIKeyLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", errors.ErrFailedToGenerateAPIKey
	}

	// Combine timestamp with random bytes
	apiKey := fmt.Sprintf("gml_%d_%s", timestamp, hex.EncodeToString(randomBytes))
	return apiKey, nil
}

// deriveKey derives a 32-byte AES key from the provided API key using SHA-256
func deriveKey(apiKey string) []byte {
	hash := sha256.Sum256([]byte(apiKey))
	return hash[:]
}

// EncryptWithAPIKey encrypts data using the provided API key
func EncryptWithAPIKey(data, apiKey string) (string, error) {
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}

	key := deriveKey(apiKey)

	// Create AES cipher
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("Failed to create AES cipher: %v", err)
		return "", errors.ErrFailedToEncryptWithAPIKey
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		log.Printf("Failed to create GCM: %v", err)
		return "", errors.ErrFailedToEncryptWithAPIKey
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		log.Printf("Failed to generate nonce: %v", err)
		return "", errors.ErrFailedToEncryptWithAPIKey
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Return base64 encoded result for easier handling
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptWithAPIKey decrypts data using the provided API key
func DecryptWithAPIKey(encryptedData, apiKey string) (string, error) {
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}

	key := deriveKey(apiKey)

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		log.Printf("Failed to decode base64: %v", err)
		return "", errors.ErrFailedToDecryptWithAPIKey
	}

	// Create AES cipher
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		log.Printf("Failed to create AES cipher: %v", err)
		return "", errors.ErrFailedToDecryptWithAPIKey
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		log.Printf("Failed to create GCM: %v", err)
		return "", errors.ErrFailedToDecryptWithAPIKey
	}

	// Extract nonce and actual ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		log.Printf("Ciphertext too short")
		return "", errors.ErrFailedToDecryptWithAPIKey
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt the data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Printf("Failed to decrypt: %v", err)
		return "", errors.ErrFailedToDecryptWithAPIKey
	}

	return string(plaintext), nil
}

// ValidateAPIKey validates if the provided API key is in correct format
func ValidateAPIKey(apiKey string) bool {
	if apiKey == "" {
		return false
	}

	// Allow the default key
	if apiKey == DefaultAPIKey {
		return true
	}

	// Check if it's a timestamped key (starts with "gml_")
	if len(apiKey) > 4 && apiKey[:4] == "gml_" {
		return true
	}

	// Check if it's a hex string of appropriate length
	if len(apiKey) == APIKeyLength*2 { // hex encoding doubles the length
		_, err := hex.DecodeString(apiKey)
		return err == nil
	}

	return false
}
