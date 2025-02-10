package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"log"
	"os"
)

func Encrypt(plaintext string) string {
	secretKey := os.Getenv("ENCRYPTION_KEY")
	aes, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		log.Println(err)
		return ""
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		log.Println(err)
		return ""
	}

	// We need a 12-byte nonce for GCM (modifiable if you use cipher.NewGCMWithNonceSize())
	// A nonce should always be randomly generated for every encryption.
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		log.Println(err)
		return ""
	}

	// ciphertext here is actually nonce+ciphertext
	// So that when we decrypt, just knowing the nonce size
	// is enough to separate it from the ciphertext.
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return string(ciphertext)
}

func Decrypt(ciphertext string) string {
	secretKey := os.Getenv("ENCRYPTION_KEY")
	aes, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		log.Println(err)
		return ""
	}

	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		log.Println(err)
		return ""
	}

	// Since we know the ciphertext is actually nonce+ciphertext
	// And len(nonce) == NonceSize(). We can separate the two.
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, []byte(nonce), []byte(ciphertext), nil)
	if err != nil {
		log.Println(err)
		return ""
	}

	return string(plaintext)
}
