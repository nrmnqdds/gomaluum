package server

import (
	"context"
	"log"
	"net/url"
	"os"

	"github.com/dgrr/cookiejar"
	"github.com/valyala/fasthttp"

	"github.com/cloudflare/cloudflare-go"
	"github.com/nrmnqdds/gomaluum/internal/constants"
	"github.com/nrmnqdds/gomaluum/internal/errors"
	auth_proto "github.com/nrmnqdds/gomaluum/internal/proto"
	cf "github.com/nrmnqdds/gomaluum/pkg/cloudflare"
)

func (s *GRPCServer) Login(ctx context.Context, props *auth_proto.LoginRequest) (*auth_proto.LoginResponse, error) {
	// Acquire cookie jar
	cj := cookiejar.AcquireCookieJar()
	defer cookiejar.ReleaseCookieJar(cj)

	req1, resp1 := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req1)
	defer fasthttp.ReleaseResponse(resp1)

	req1.SetRequestURI(constants.ImaluumCasPage)

	if err := fasthttp.Do(req1, resp1); err != nil {
		log.Printf("error: %v", err)
		return nil, errors.ErrURLParseFailed
	}

	// Capture cookie from response 1
	cj.ReadResponse(resp1)

	req2, resp2 := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req2)
	defer fasthttp.ReleaseResponse(resp2)

	formVal := url.Values{
		"username":    {props.Username},
		"password":    {props.Password},
		"execution":   {"e1s1"},
		"_eventId":    {"submit"},
		"geolocation": {""},
	}.Encode()

	req2.SetRequestURI(constants.ImaluumLoginPage)
	req2.Header.SetMethod("POST")
	req2.Header.SetContentType("application/x-www-form-urlencoded")
	req2.SetBodyString(formVal)
	for {
		// Read cookie by cookie
		c := cj.Get()
		if c == nil {
			break
		}
		req2.Header.SetCookieBytesKV(c.Key(), c.Value())
		fasthttp.ReleaseCookie(c)
	}

	if err := fasthttp.Do(req2, resp2); err != nil {
		log.Printf("error: %v", err)
		return nil, errors.ErrURLParseFailed
	}

	// Capture cookie from response 2
	cj.ReadResponse(resp2)

	location := resp2.Header.Peek("Location")

	req3, resp3 := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req3)
	defer fasthttp.ReleaseResponse(resp3)

	req3.SetRequestURI(string(location))
	for {
		// Read cookie by cookie
		c := cj.Get()
		if c == nil {
			break
		}
		req3.Header.SetCookieBytesKV(c.Key(), c.Value())
		fasthttp.ReleaseCookie(c)
	}

	if err := fasthttp.Do(req3, resp3); err != nil {
		log.Printf("error: %v", err)
		return nil, errors.ErrURLParseFailed
	}

	// Capture cookie from response 3
	cj.ReadResponse(resp3)

	for {
		// Read cookie by cookie
		c := cj.Get()
		if c == nil {
			break
		}

		if string(c.Key()) == "MOD_AUTH_CAS" {
			cookie := string(c.Value())
			go func() {
				if err := SaveToKV(ctx, props.Username, props.Password); err != nil {
					log.Printf("Failed to save to KV: %v", err)
				}
			}()

			return &auth_proto.LoginResponse{
				Token:    cookie,
				Username: props.Username,
				Password: props.Password,
			}, nil
		}
		fasthttp.ReleaseCookie(c)
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
