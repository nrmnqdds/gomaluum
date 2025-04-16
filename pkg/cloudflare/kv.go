package cloudflare

import (
	"context"
	"log"
	"os"

	"github.com/cloudflare/cloudflare-go"
)

type AppCloudflare struct {
	*cloudflare.API
}

func New() *AppCloudflare {
	cloudflareClient, err := cloudflare.NewWithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))
	if err != nil {
		log.Fatalf("Error initiating cloudflare client: %s", err.Error())
	}

	return &AppCloudflare{
		API: cloudflareClient,
	}
}

func (s *AppCloudflare) GetClient() *cloudflare.API {
	return s.API
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

	cfClient := New()
	client := cfClient.GetClient()

	if _, err := client.WriteWorkersKVEntry(ctx, kvResourceContainer, kvEntryParams); err != nil {
		return err
	}

	return nil
}
