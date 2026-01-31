# gRPC Services Configuration

This document explains how the application connects to multiple gRPC services and how to add new services.

## Current Services

### GAS (Gomaluum Auth Service)
- **Purpose**: Authentication service
- **Environment Variable**: `GAS_SERVICE_URL`
- **Proto Client**: `auth_proto.AuthClient`
- **Usage**: Used for login and token refresh operations

### GEI (Gomaluum Event Indexer)
- **Purpose**: Schedule indexer service
- **Environment Variable**: `GEI_SERVICE_URL`
- **Proto Client**: To be implemented when proto definition is available
- **Usage**: To be determined based on schedule indexing requirements

## Environment Variables

Add these to your `.env` file:

```env
# gRPC Services
GAS_SERVICE_URL=localhost:50051
GEI_SERVICE_URL=localhost:50052
```

## Architecture

### GRPCClients Struct
The `GRPCClients` struct in `internal/server/server.go` holds all gRPC service connections:

```go
type GRPCClients struct {
    // GAS - Gomaluum Auth Service
    GASConn   *grpc.ClientConn
    GASClient auth_proto.AuthClient

    // GEI - Gomaluum Event Indexer
    GEIConn *grpc.ClientConn
    // GEIClient will be added when proto definition is available
}
```

### Service Configuration
Services are configured using the `GRPCServiceConfig` struct:

```go
type GRPCServiceConfig struct {
    Name string  // Service identifier (e.g., "GAS", "GEI")
    URL  string  // Service URL (e.g., "localhost:50051")
}
```

### Initialization
All services are initialized in `main.go`:

```go
grpcConfigs := []server.GRPCServiceConfig{
    {Name: "GAS", URL: gasServiceURL},
    {Name: "GEI", URL: geiServiceURL},
}

grpcClients, err := server.NewGRPCClients(grpcConfigs)
```

## Adding a New gRPC Service

Follow these steps to add a new gRPC service (example: "GXX"):

### 1. Add Proto Definition
Create or import the proto files for your service in `internal/proto/`.

### 2. Update GRPCClients Struct
In `internal/server/server.go`, add fields for the new service:

```go
type GRPCClients struct {
    // ... existing services ...
    
    // GXX - Your New Service
    GXXConn   *grpc.ClientConn
    GXXClient xxx_proto.XxxClient
}
```

### 3. Update NewGRPCClients Function
Add a case in the switch statement in `NewGRPCClients()`:

```go
switch config.Name {
    // ... existing cases ...
    
    case "GXX":
        clients.GXXConn = conn
        clients.GXXClient = xxx_proto.NewXxxClient(conn)
        log.Printf("Connected to GXX at %s", config.URL)
}
```

### 4. Update Close Method
Add cleanup for the new connection in the `Close()` method:

```go
func (c *GRPCClients) Close() error {
    // ... existing closes ...
    
    if c.GXXConn != nil {
        if err := c.GXXConn.Close(); err != nil {
            log.Printf("Error closing GXX connection: %v", err)
            lastErr = err
        }
    }
    
    return lastErr
}
```

### 5. Update main.go
Add environment variable and configuration:

```go
// Get service URL
gxxServiceURL := os.Getenv("GXX_SERVICE_URL")
if gxxServiceURL == "" {
    log.Fatal("GXX_SERVICE_URL environment variable is required")
}

// Add to configs
grpcConfigs := []server.GRPCServiceConfig{
    // ... existing services ...
    {Name: "GXX", URL: gxxServiceURL},
}
```

Update the console output:

```go
fmt.Println(gchalk.Green("Connected to gRPC services:"))
fmt.Println(gchalk.Green(fmt.Sprintf("  • GAS (Auth Service): %s", gasServiceURL)))
fmt.Println(gchalk.Green(fmt.Sprintf("  • GEI (Schedule Indexer): %s", geiServiceURL)))
fmt.Println(gchalk.Green(fmt.Sprintf("  • GXX (Your Service): %s", gxxServiceURL)))
```

### 6. Update Environment Variables
Add to your `.env` file:

```env
GXX_SERVICE_URL=localhost:50053
```

### 7. Use the Client
In your handlers, access the client via:

```go
resp, err := s.grpcClients.GXXClient.YourMethod(ctx, req)
```

## Usage Examples

### Using GAS Client (Auth)
```go
// In a handler
resp, err := s.grpcClients.GASClient.Login(ctx, &pb.LoginRequest{
    Username: username,
    Password: password,
})
```

### Using GEI Client (When Implemented)
```go
// In a handler
resp, err := s.grpcClients.GEIClient.GetSchedule(ctx, &schedule_pb.ScheduleRequest{
    StudentId: studentId,
})
```

## Error Handling

The `NewGRPCClients()` function:
- Validates each service connection
- Closes any successful connections if a later connection fails
- Returns descriptive errors indicating which service failed

Example error:
```
failed to connect to GEI gRPC service at localhost:50052: connection refused
```

## Graceful Shutdown

All gRPC connections are properly closed during graceful shutdown:
1. `main.go` defers `grpcClients.Close()`
2. The `Close()` method closes all active connections
3. Errors are logged but don't prevent other connections from closing

## Best Practices

1. **Service Naming**: Use uppercase 3-letter acronyms (GAS, GEI, etc.)
2. **Environment Variables**: Follow the pattern `{SERVICE}_SERVICE_URL`
3. **Error Handling**: Always check for connection errors during startup
4. **Logging**: Log successful connections and connection errors
5. **Proto Organization**: Keep proto files organized by service in `internal/proto/`
6. **Documentation**: Update this file when adding new services

## Troubleshooting

### Connection Refused
- Ensure the gRPC service is running
- Verify the URL in your `.env` file
- Check firewall/network settings

### Missing Environment Variable
- Verify the variable is set in your `.env` file (dev mode)
- Verify the variable is set in your environment (production mode)
- Check for typos in variable names

### Client Method Not Available
- Ensure proto files are up to date
- Regenerate proto files if needed
- Verify the client type is correctly assigned in `NewGRPCClients()`
