package server

import (
	"context"
	"log"
	"time"

	"github.com/cristalhq/base64"

	"aidanwoods.dev/go-paseto"
	pb "github.com/nrmnqdds/gomaluum/internal/proto"
	"github.com/nrmnqdds/gomaluum/pkg/apikey"
)

type TokenPayload struct {
	username      string
	password      string
	imaluumCookie string
	apiKey        string
}

// GeneratePasetoToken generates a PASETO token for the given original uia cookie
// origin: the original uia cookie
// username: the username of the user
// password: the password of the user in base64
func (s *Server) GeneratePasetoToken(payload TokenPayload) (string, string, error) {
	logger := s.log.GetLogger()
	token := s.paseto.Token

	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	// token.SetExpiration(time.Now().Add(time.Minute * 30)) // 30 minutes
	token.SetExpiration(time.Now()) // now
	// token.SetExpiration(time.Now().Add(time.Minute * 1)) // 1 minutes
	token.SetIssuer("gomaluum")

	originPassword := payload.password
	imaluumCookie := payload.imaluumCookie
	username := payload.username
	userAPIKey := payload.apiKey

	// encode the base64 password
	password := []byte(originPassword)
	base64Password := base64.StdEncoding.EncodeToString(password)

	// Encrypt sensitive data with API key before storing in PASETO
	encryptedCookie, err := apikey.EncryptWithAPIKey(imaluumCookie, userAPIKey)
	if err != nil {
		logger.Sugar().Errorf("Failed to encrypt cookie with API key: %v", err)
		return "", "", err
	}

	encryptedUsername, err := apikey.EncryptWithAPIKey(username, userAPIKey)
	if err != nil {
		logger.Sugar().Errorf("Failed to encrypt username with API key: %v", err)
		return "", "", err
	}

	encryptedPassword, err := apikey.EncryptWithAPIKey(base64Password, userAPIKey)
	if err != nil {
		logger.Sugar().Errorf("Failed to encrypt password with API key: %v", err)
		return "", "", err
	}

	token.SetString("imaluumCookie", encryptedCookie)
	token.SetString("username", encryptedUsername)
	token.SetString("password", encryptedPassword)

	signed := token.V4Sign(*s.paseto.PrivateKey, nil)

	s.paseto.Token = token

	return signed, imaluumCookie, nil
}

// DecodePasetoToken decodes the given PASETO token and returns the original uia cookie
func (s *Server) DecodePasetoToken(token, userAPIKey string) (*TokenPayload, error) {
	parser := paseto.NewParserWithoutExpiryCheck() // Don't use NewParser() which will checks expiry by default
	logger := s.log.GetLogger()

	ctx := context.Background()

	// Don't throw an error immediately if the token has expired
	// parser.AddRule(paseto.NotExpired())         // this will fail if the token has expired
	parser.AddRule(paseto.IssuedBy("gomaluum")) // this will fail if the token was not issued by "gomaluum"

	decodedToken, err := parser.ParseV4Public(*s.paseto.PublicKey, token, nil) // this will fail if parsing failes, cryptographic checks fail, or validation rules fail
	if err != nil {
		logger.Sugar().Errorf("Failed to parse token: %v", err)

		return nil, err
	}

	tokenExpiryDate, err := decodedToken.GetExpiration()
	if err != nil {
		logger.Sugar().Errorf("Failed to get expiration: %v", err)
		return nil, err
	}

	today := time.Now()

	// Get encrypted data from token
	encryptedUsername, _ := decodedToken.GetString("username")
	encryptedPassword, _ := decodedToken.GetString("password")

	// Decrypt data using API key
	username, err := apikey.DecryptWithAPIKey(encryptedUsername, userAPIKey)
	if err != nil {
		logger.Sugar().Errorf("Failed to decrypt username with API key: %v", err)
		return nil, err
	}

	password, err := apikey.DecryptWithAPIKey(encryptedPassword, userAPIKey)
	if err != nil {
		logger.Sugar().Errorf("Failed to decrypt password with API key: %v", err)
		return nil, err
	}

	// if the token has expired, we need to regenerate it
	if today.After(tokenExpiryDate) {
		logger.Info("Token has expired")

		// decode the password
		decodedPassword, err := base64.StdEncoding.DecodeString(password)
		if err != nil {
			logger.Sugar().Errorf("Failed to decode password: %v", err)
			return nil, err
		}

		refresh := func() (string, time.Time, error) {
			// regenerate the token
			logger.Sugar().Infof("Refreshing session token with username: %s, password: %s", username, string(decodedPassword))

			resp, err := s.grpc.client.Login(ctx, &pb.LoginRequest{
				Username: username,
				Password: string(decodedPassword),
			})

			if err != nil {
				logger.Sugar().Errorf("Failed to login: %v", err)
				return "", time.Now(), err
			}

			return resp.Token, time.Now(), nil
		}

		newToken, err := s.tokenManager.GetToken(username, refresh)
		if err != nil {
			logger.Sugar().Errorf("Failed to get token: %v", err)
			log.Fatal(err)
		}

		logger.Sugar().Infof("Refreshed token: %s with origin for user: %s", newToken, username)

		go s.UpdateAnalytics(username)

		return &TokenPayload{
			username:      username,
			password:      string(decodedPassword),
			imaluumCookie: newToken,
			apiKey:        userAPIKey,
		}, nil

		// End of if token expired
	}

	// If token not expired yet - decrypt the cookie
	encryptedCookie, _ := decodedToken.GetString("imaluumCookie")
	imaluumCookie, err := apikey.DecryptWithAPIKey(encryptedCookie, userAPIKey)
	if err != nil {
		logger.Sugar().Errorf("Failed to decrypt cookie with API key: %v", err)
		return nil, err
	}

	go s.UpdateAnalytics(username)
	return &TokenPayload{
		username:      username,
		password:      password,
		imaluumCookie: imaluumCookie,
		apiKey:        userAPIKey,
	}, nil
}
