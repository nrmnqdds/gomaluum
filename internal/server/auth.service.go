package server

import (
	"context"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
	cf "github.com/nrmnqdds/gomaluum/pkg/cloudflare"
)

func (s *GRPCServer) Login(ctx context.Context, req *auth_proto.LoginRequest) (*auth_proto.LoginResponse, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.ErrCookieJarCreationFailed
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: time.Second * 10, // Indicates i-Ma'luum server is slow
	}

	urlObj, err := url.Parse(constants.ImaluumPage)
	if err != nil {
		return nil, errors.ErrURLParseFailed
	}

	formVal := url.Values{
		"username":    {req.Username},
		"password":    {req.Password},
		"execution":   {"e1s1"},
		"_eventId":    {"submit"},
		"geolocation": {""},
	}

	// First request
	reqFirst, err := http.NewRequest("GET", constants.ImaluumCasPage, nil)
	if err != nil {
		reqFirst.Body.Close()
		return nil, errors.ErrURLParseFailed
	}

	setHeaders(reqFirst)

	respFirst, err := client.Do(reqFirst)
	if err != nil {
		// reqFirst.Body.Close()
		// respFirst.Body.Close()
		return nil, errors.ErrURLParseFailed
	}
	// if err := reqFirst.Body.Close(); err != nil {
	// 	log.Printf("Failed to close request body: %v", err)
	// 	return nil, errors.ErrURLParseFailed
	// }
	// if err := respFirst.Body.Close(); err != nil {
	// 	log.Printf("Failed to close response body: %v", err)
	// 	return nil, errors.ErrURLParseFailed
	// }

	client.Jar.SetCookies(urlObj, respFirst.Cookies())

	// Second request
	reqSecond, err := http.NewRequest("POST", constants.ImaluumLoginPage, strings.NewReader(formVal.Encode()))
	if err != nil {
		reqSecond.Body.Close()
		return nil, errors.ErrURLParseFailed
	}
	reqSecond.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setHeaders(reqSecond)

	if _, err := client.Do(reqSecond); err != nil {
		// reqSecond.Body.Close()
		return nil, errors.ErrURLParseFailed
	}
	// if err != nil {
	// 	reqSecond.Body.Close()
	// 	respSecond.Body.Close()
	// 	return nil, errors.ErrURLParseFailed
	// }
	// respSecond.Body.Close()

	cookies := client.Jar.Cookies(urlObj)

	for _, cookie := range cookies {
		if cookie.Name == "MOD_AUTH_CAS" {

			// Save the username and password to KV
			// Use goroutine to avoid blocking the main thread
			// go SaveToKV(req.Username, req.Password)
			go func() {
				if err := SaveToKV(ctx, req.Username, req.Password); err != nil {
					log.Printf("Failed to save to KV: %v", err)
				}
			}()

			resp := &auth_proto.LoginResponse{
				Token:    cookie.Value,
				Username: req.Username,
				Password: req.Password,
			}

			return resp, nil

		}
	}

	return nil, errors.ErrLoginFailed
}

func SaveToKV(ctx context.Context, username, password string) error {
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

	cfClient := cf.New()
	client := cfClient.GetClient()

	if _, err := client.WriteWorkersKVEntry(ctx, kvResourceContainer, kvEntryParams); err != nil {
		return err
	}

	return nil
}
