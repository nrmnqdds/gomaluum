package server

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	"github.com/nrmnqdds/gomaluum/internal/proto"
)

// Login is a GRPC function to authenticate the user
// Returns CAS cookie, username, and password
func (s *Server) Login(props *proto.LoginRequest) (*proto.LoginResponse, error) {
	jar, _ := cookiejar.New(nil)

	logger := s.log.GetLogger()

	client := &http.Client{
		Jar: jar,
	}

	urlObj, err := url.Parse(constants.ImaluumPage)
	if err != nil {
		logger.Sugar().Errorf("Failed to parse url: %v", err)
		return nil, errors.ErrURLParseFailed
	}

	formVal := url.Values{
		"username":    {props.Username},
		"password":    {props.Password},
		"execution":   {"e1s1"},
		"_eventId":    {"submit"},
		"geolocation": {""},
	}

	// First request
	logger.Debug("Making first request")
	reqFirst, _ := http.NewRequest("GET", constants.ImaluumCasPage, nil)
	setHeaders(reqFirst)

	respFirst, err := client.Do(reqFirst)
	if err != nil {
		logger.Sugar().Errorf("Failed to login first request: %v", err)
		return nil, errors.ErrURLParseFailed
	}
	respFirst.Body.Close()

	client.Jar.SetCookies(urlObj, respFirst.Cookies())

	// Second request
	logger.Debug("Making second request")
	reqSecond, _ := http.NewRequest("POST", constants.ImaluumLoginPage, strings.NewReader(formVal.Encode()))
	reqSecond.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setHeaders(reqSecond)

	respSecond, err := client.Do(reqSecond)
	if err != nil {
		logger.Sugar().Errorf("Failed to login second request: %v", err)
		return nil, errors.ErrURLParseFailed
	}
	respSecond.Body.Close()

	cookies := client.Jar.Cookies(urlObj)

	for _, cookie := range cookies {
		logger.Sugar().Debugf("Cookie: %v", cookie)
		if cookie.Name == "MOD_AUTH_CAS" {

			// Save the username and password to KV
			// Use goroutine to avoid blocking the main thread
			go s.SaveToKV(props.Username, props.Password)

			resp := &proto.LoginResponse{
				Token:    cookie.Value,
				Username: props.Username,
				Password: props.Password,
			}

			return resp, nil

		}
	}

	return nil, errors.ErrLoginFailed
}

func (s *Server) SaveToKV(username, password string) {
	ctx := context.Background()

	kvEntryParams := cloudflare.WriteWorkersKVEntryParams{
		NamespaceID: os.Getenv("KV_NAMESPACE_ID"),
		Key:         username,
		Value:       []byte(password),
	}

	kvResourceContainer := &cloudflare.ResourceContainer{
		Level:      "accounts",
		Identifier: os.Getenv("KV_USER_ID"),
		Type:       "account",
	}

	cfClient := s.cf.GetClient()

	_, cerr := cfClient.WriteWorkersKVEntry(ctx, kvResourceContainer, kvEntryParams)
	if cerr != nil {
		s.log.Sugar().Errorf("Failed to write to KV: %v", cerr)
	}
	s.log.Sugar().Debugf("Successfully wrote to KV")
}

// Function to set headers for a request.
func setHeaders(req *http.Request) {
	req.Header.Set("Connection", "Keep-Alive")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("User-Agent", "Mozilla/5.0")
}
