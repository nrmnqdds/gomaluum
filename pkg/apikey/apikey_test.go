package apikey

import (
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(key) != APIKeyLength*2 { // hex encoding doubles the length
		t.Errorf("Expected key length %d, got %d", APIKeyLength*2, len(key))
	}

	// Generate another key to ensure they're different
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if key == key2 {
		t.Error("Expected different keys, got identical ones")
	}
}

func TestGenerateTimestampedAPIKey(t *testing.T) {
	key, err := GenerateTimestampedAPIKey()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(key) < 4 || key[:4] != "gml_" {
		t.Errorf("Expected timestamped key to start with 'gml_', got %s", key[:4])
	}

	// Generate another key to ensure they're different
	key2, err := GenerateTimestampedAPIKey()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if key == key2 {
		t.Error("Expected different timestamped keys, got identical ones")
	}
}

func TestEncryptDecryptWithAPIKey(t *testing.T) {
	testData := "This is sensitive test data"
	apiKey := "test-api-key-for-encryption"

	// Test encryption
	encrypted, err := EncryptWithAPIKey(testData, apiKey)
	if err != nil {
		t.Fatalf("Expected no error during encryption, got %v", err)
	}

	if encrypted == testData {
		t.Error("Expected encrypted data to be different from original")
	}

	// Test decryption
	decrypted, err := DecryptWithAPIKey(encrypted, apiKey)
	if err != nil {
		t.Fatalf("Expected no error during decryption, got %v", err)
	}

	if decrypted != testData {
		t.Errorf("Expected decrypted data to match original. Got %s, want %s", decrypted, testData)
	}
}

func TestEncryptDecryptWithDefaultKey(t *testing.T) {
	testData := "Test data with default key"

	// Test with empty API key (should use default)
	encrypted, err := EncryptWithAPIKey(testData, "")
	if err != nil {
		t.Fatalf("Expected no error during encryption with default key, got %v", err)
	}

	decrypted, err := DecryptWithAPIKey(encrypted, "")
	if err != nil {
		t.Fatalf("Expected no error during decryption with default key, got %v", err)
	}

	if decrypted != testData {
		t.Errorf("Expected decrypted data to match original. Got %s, want %s", decrypted, testData)
	}

	// Test explicit default key
	encrypted2, err := EncryptWithAPIKey(testData, DefaultAPIKey)
	if err != nil {
		t.Fatalf("Expected no error during encryption with explicit default key, got %v", err)
	}

	decrypted2, err := DecryptWithAPIKey(encrypted2, DefaultAPIKey)
	if err != nil {
		t.Fatalf("Expected no error during decryption with explicit default key, got %v", err)
	}

	if decrypted2 != testData {
		t.Errorf("Expected decrypted data to match original. Got %s, want %s", decrypted2, testData)
	}
}

func TestEncryptDecryptWithWrongKey(t *testing.T) {
	testData := "This should fail with wrong key"
	apiKey := "correct-key"
	wrongKey := "wrong-key"

	// Encrypt with correct key
	encrypted, err := EncryptWithAPIKey(testData, apiKey)
	if err != nil {
		t.Fatalf("Expected no error during encryption, got %v", err)
	}

	// Try to decrypt with wrong key (should fail)
	_, err = DecryptWithAPIKey(encrypted, wrongKey)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key, got nil")
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{"Empty key", "", false},
		{"Default key", DefaultAPIKey, true},
		{"Valid hex key", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", true},
		{"Invalid hex key", "xyz123", false},
		{"Valid timestamped key", "gml_1703123456_abcdef1234567890", true},
		{"Invalid timestamped key", "invalid_1703123456_abcdef", false},
		{"Too short hex", "abc123", false},
		{"Too long hex", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateAPIKey(tt.key)
			if result != tt.expected {
				t.Errorf("ValidateAPIKey(%s) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestDifferentKeysProduceDifferentResults(t *testing.T) {
	testData := "Same data, different keys"
	key1 := "key-one"
	key2 := "key-two"

	encrypted1, err := EncryptWithAPIKey(testData, key1)
	if err != nil {
		t.Fatalf("Expected no error with key1, got %v", err)
	}

	encrypted2, err := EncryptWithAPIKey(testData, key2)
	if err != nil {
		t.Fatalf("Expected no error with key2, got %v", err)
	}

	if encrypted1 == encrypted2 {
		t.Error("Expected different encryption results for different keys")
	}

	// Verify each can be decrypted with its own key
	decrypted1, err := DecryptWithAPIKey(encrypted1, key1)
	if err != nil || decrypted1 != testData {
		t.Errorf("Failed to decrypt with key1: err=%v, data=%s", err, decrypted1)
	}

	decrypted2, err := DecryptWithAPIKey(encrypted2, key2)
	if err != nil || decrypted2 != testData {
		t.Errorf("Failed to decrypt with key2: err=%v, data=%s", err, decrypted2)
	}
}

func TestMultipleEncryptionsAreDifferent(t *testing.T) {
	testData := "Same data, same key, should produce different ciphertexts"
	apiKey := "test-key"

	encrypted1, err := EncryptWithAPIKey(testData, apiKey)
	if err != nil {
		t.Fatalf("Expected no error in first encryption, got %v", err)
	}

	encrypted2, err := EncryptWithAPIKey(testData, apiKey)
	if err != nil {
		t.Fatalf("Expected no error in second encryption, got %v", err)
	}

	if encrypted1 == encrypted2 {
		t.Error("Expected different ciphertexts for same data due to random nonces")
	}

	// Both should decrypt to the same original data
	decrypted1, err := DecryptWithAPIKey(encrypted1, apiKey)
	if err != nil || decrypted1 != testData {
		t.Errorf("Failed to decrypt first ciphertext: err=%v, data=%s", err, decrypted1)
	}

	decrypted2, err := DecryptWithAPIKey(encrypted2, apiKey)
	if err != nil || decrypted2 != testData {
		t.Errorf("Failed to decrypt second ciphertext: err=%v, data=%s", err, decrypted2)
	}
}
