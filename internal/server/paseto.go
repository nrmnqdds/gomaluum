package server

import (
	"context"
	"time"

	"github.com/cristalhq/base64"

	"aidanwoods.dev/go-paseto"
	pb "github.com/nrmnqdds/gomaluum/internal/proto"
)

// GeneratePasetoToken generates a PASETO token for the given original uia cookie
// origin: the original uia cookie
// username: the username of the user
// password: the password of the user in base64
func (s *Server) GeneratePasetoToken(origin, username, originPassword string) (string, string, error) {
	token := s.paseto.Token

	token.SetIssuedAt(time.Now())
	token.SetNotBefore(time.Now())
	// token.SetExpiration(time.Now().Add(time.Minute * 30)) // 30 minutes
	token.SetExpiration(time.Now()) // now
	// token.SetExpiration(time.Now().Add(time.Minute * 1)) // 1 minutes
	token.SetIssuer("gomaluum")

	// encode the base64 password
	password := []byte(originPassword)
	base64Password := base64.StdEncoding.EncodeToString(password)

	token.SetString("origin", origin)
	token.SetString("username", username)
	token.SetString("password", base64Password)

	signed := token.V4Sign(*s.paseto.PrivateKey, nil)

	s.paseto.Token = token

	return signed, origin, nil
}

// DecodePasetoToken decodes the given PASETO token and returns the original uia cookie
func (s *Server) DecodePasetoToken(token string) (string, error) {
	parser := paseto.NewParserWithoutExpiryCheck() // Don't use NewParser() which will checks expiry by default
	logger := s.log.GetLogger()

	ctx := context.Background()

	// Don't throw an error immediately if the token has expired
	// parser.AddRule(paseto.NotExpired())         // this will fail if the token has expired
	parser.AddRule(paseto.IssuedBy("gomaluum")) // this will fail if the token was not issued by "gomaluum"

	decodedToken, err := parser.ParseV4Public(*s.paseto.PublicKey, token, nil) // this will fail if parsing failes, cryptographic checks fail, or validation rules fail
	if err != nil {
		logger.Sugar().Errorf("Failed to parse token: %v", err)

		return "", err
	}

	tokenExpiryDate, err := decodedToken.GetExpiration()
	if err != nil {
		logger.Sugar().Errorf("Failed to get expiration: %v", err)
		return "", err
	}

	today := time.Now()

	// if the token has expired, we need to regenerate it
	if today.After(tokenExpiryDate) {
		logger.Info("Token has expired")

		username, _ := decodedToken.GetString("username")
		password, _ := decodedToken.GetString("password")

		// decode the password
		decodedPassword, err := base64.StdEncoding.DecodeString(password)
		if err != nil {
			logger.Sugar().Errorf("Failed to decode password: %v", err)
			return "", err
		}
		// regenerate the token
		logger.Sugar().Infof("Refreshing session token with username: %s, password: %s", username, string(decodedPassword))

		resp, err := s.grpc.Login(ctx, &pb.LoginRequest{
			Username: username,
			Password: string(decodedPassword),
		})

		if err != nil {
			logger.Sugar().Errorf("Failed to login: %v", err)
			return "", err
		}

		logger.Sugar().Infof("Regenerated token: %s with origin for user: %s", resp.Token, username)
		return resp.Token, nil
	}

	origin, err := decodedToken.GetString("origin")
	if err != nil {
		return "", err
	}

	return origin, nil
}
