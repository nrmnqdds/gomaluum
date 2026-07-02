package server

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/nrmnqdds/gomaluum/internal/dtos"
	gei "github.com/nrmnqdds/gomaluum/internal/proto/gei"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// scheduleIndexer is the client for GEI's ScheduleIndexer: gomaluum's shared,
// encrypted cache for scraped schedules. It persists across the stateless
// serverless instances that in-process caches cannot span, letting a request
// re-scrape only the latest semester and serve the rest from cache.
type scheduleIndexer struct {
	conn     *grpc.ClientConn
	client   gei.ScheduleIndexerClient
	adminKey string
}

// newScheduleIndexer dials GEI and returns a client. adminKey guards writes.
func newScheduleIndexer(serviceURL, adminKey string) (*scheduleIndexer, error) {
	conn, err := grpc.NewClient(serviceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GEI at %s: %w", serviceURL, err)
	}
	return &scheduleIndexer{
		conn:     conn,
		client:   gei.NewScheduleIndexerClient(conn),
		adminKey: adminKey,
	}, nil
}

func (i *scheduleIndexer) Close() error {
	if i.conn != nil {
		return i.conn.Close()
	}
	return nil
}

// GetSchedule returns the cached schedule for username. found is false (with a
// nil error) when the user has nothing cached yet.
func (i *scheduleIndexer) GetSchedule(ctx context.Context, username string) (schedules []dtos.ScheduleResponse, found bool, err error) {
	resp, err := i.client.GetSchedule(ctx, &gei.GetScheduleRequest{Username: username})
	if err != nil {
		return nil, false, err
	}
	if !resp.GetSuccess() {
		return nil, false, nil
	}
	if err := sonic.ConfigFastest.Unmarshal([]byte(resp.GetScheduleJson()), &schedules); err != nil {
		return nil, false, fmt.Errorf("decoding cached schedule: %w", err)
	}
	return schedules, true, nil
}

// StoreSchedule caches the schedule for username. The admin key is attached as
// request metadata because GEI guards writes.
func (i *scheduleIndexer) StoreSchedule(ctx context.Context, username string, schedules []dtos.ScheduleResponse) error {
	payload, err := sonic.ConfigFastest.Marshal(schedules)
	if err != nil {
		return fmt.Errorf("encoding schedule for cache: %w", err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "admin-key", i.adminKey)
	_, err = i.client.StoreSchedule(ctx, &gei.StoreScheduleRequest{
		Username:     username,
		ScheduleJson: string(payload),
	})
	return err
}
