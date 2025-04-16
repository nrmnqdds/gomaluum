package server

import (
	"context"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
)

func (s *GRPCServer) Login(_ context.Context, req *auth_proto.LoginRequest) (*auth_proto.LoginResponse, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Printf("Failed to create cookie jar: %v", err)
		return nil, errors.ErrCookieJarCreationFailed
	}

	client := &http.Client{
		Transport: s.httpClient.Transport,
		Jar:       jar,
		Timeout:   time.Second * 10, // Indicates i-Ma'luum server is slow
	}

	urlObj, err := url.Parse(constants.ImaluumPage)
	if err != nil {
		log.Printf("Failed to parse Imaluum Page: %v", err)
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
		log.Printf("Failed to create first request: %v", err)
		if err := reqFirst.Body.Close(); err != nil {
			log.Printf("Failed to close request body: %v", err)
			return nil, errors.ErrFailedToCloseRequestBody
		}
		return nil, errors.ErrURLParseFailed
	}

	setHeaders(reqFirst)

	respFirst, err := client.Do(reqFirst)
	if err != nil {
		log.Printf("Failed to do first request: %v", err)
		if err := reqFirst.Body.Close(); err != nil {
			log.Printf("Failed to close request body: %v", err)
			return nil, errors.ErrFailedToCloseRequestBody
		}
		if err := respFirst.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
			return nil, errors.ErrFailedToCloseResponseBody
		}
		return nil, errors.ErrURLParseFailed
	}

	client.Jar.SetCookies(urlObj, respFirst.Cookies())

	// Second request
	reqSecond, err := http.NewRequest("POST", constants.ImaluumLoginPage, strings.NewReader(formVal.Encode()))
	if err != nil {
		log.Printf("Failed to create second request: %v", err)
		if err := reqSecond.Body.Close(); err != nil {
			log.Printf("Failed to close request body: %v", err)
			return nil, errors.ErrFailedToCloseRequestBody
		}
		return nil, errors.ErrURLParseFailed
	}
	reqSecond.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setHeaders(reqSecond)

	respSecond, err := client.Do(reqSecond)
	if err != nil {
		log.Printf("Failed to do second request: %v", err)
		if err := reqSecond.Body.Close(); err != nil {
			log.Printf("Failed to close request body: %v", err)
			return nil, errors.ErrFailedToCloseRequestBody
		}
		if err := respSecond.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
			return nil, errors.ErrFailedToCloseResponseBody
		}
		return nil, errors.ErrURLParseFailed
	}
	if err := respSecond.Body.Close(); err != nil {
		log.Printf("Failed to close response body: %v", err)
		return nil, errors.ErrFailedToCloseResponseBody
	}

	cookies := client.Jar.Cookies(urlObj)

	for _, cookie := range cookies {
		if cookie.Name == "MOD_AUTH_CAS" {

			// Save the username and password to KV for caching purpose
			// Use goroutine to avoid blocking the main thread
			// go func() {
			// 	if err := SaveToKV(ctx, req.Username, req.Password); err != nil {
			// 		log.Printf("Failed to save to KV: %v", err)
			// 	}
			// }()

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
