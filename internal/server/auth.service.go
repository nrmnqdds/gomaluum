package server

import (
	"context"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
	cf "github.com/nrmnqdds/gomaluum/pkg/cloudflare"
)

func (s *GRPCServer) Login(ctx context.Context, req *auth_proto.LoginRequest) (*auth_proto.LoginResponse, error) {
	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: jar,
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
	reqFirst, _ := http.NewRequest("GET", constants.ImaluumCasPage, nil)
	setHeaders(reqFirst)

	respFirst, err := client.Do(reqFirst)
	if err != nil {
		return nil, errors.ErrURLParseFailed
	}
	respFirst.Body.Close()

	client.Jar.SetCookies(urlObj, respFirst.Cookies())

	// Second request
	reqSecond, _ := http.NewRequest("POST", constants.ImaluumLoginPage, strings.NewReader(formVal.Encode()))
	reqSecond.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setHeaders(reqSecond)

	respSecond, err := client.Do(reqSecond)
	if err != nil {
		return nil, errors.ErrURLParseFailed
	}
	respSecond.Body.Close()

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
