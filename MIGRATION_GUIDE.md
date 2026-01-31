# Migration Guide: Single to Multiple gRPC Services

## Overview
This guide helps you migrate from the old single gRPC service configuration to the new multi-service architecture.

## What Changed?

### Before (Single Service)
```go
// Environment variable
GRPC_SERVICE_URL=localhost:50051

// Code structure
type GRPCClient struct {
    conn   *grpc.ClientConn
    client auth_proto.AuthClient
}

// Initialization
grpcClient, err := server.NewGRPCClient(grpcServiceURL)
```

### After (Multiple Services)
```go
// Environment variables
GAS_SERVICE_URL=localhost:50051  // Auth Service
GEI_SERVICE_URL=localhost:50052  // Schedule Indexer

// Code structure
type GRPCClients struct {
    GASConn   *grpc.ClientConn
    GASClient auth_proto.AuthClient
    
    GEIConn   *grpc.ClientConn
    // GEIClient will be added later
}

// Initialization
grpcConfigs := []server.GRPCServiceConfig{
    {Name: "GAS", URL: gasServiceURL},
    {Name: "GEI", URL: geiServiceURL},
}
grpcClients, err := server.NewGRPCClients(grpcConfigs)
```

## Migration Steps

### Step 1: Update Environment Variables

#### Development (.env file)
```diff
- GRPC_SERVICE_URL=localhost:50051
+ GAS_SERVICE_URL=localhost:50051
+ GEI_SERVICE_URL=localhost:50052
```

#### Production (Environment)
Update your deployment configuration to include both variables:
```bash
export GAS_SERVICE_URL=your-gas-service:50051
export GEI_SERVICE_URL=your-gei-service:50052
```

### Step 2: Update Code References

If you have any custom code that references the old gRPC client:

```diff
- resp, err := s.grpc.client.Login(ctx, user)
+ resp, err := s.grpcClients.GASClient.Login(ctx, user)
```

### Step 3: Test Your Application

1. **Start your gRPC services**
   ```bash
   # Terminal 1: Start GAS service
   cd gomaluum-auth-service
   ./gas-server -p 50051

   # Terminal 2: Start GEI service
   cd gomaluum-event-indexer
   ./gei-server -p 50052
   ```

2. **Start your application**
   ```bash
   go run main.go -d  # Development mode
   ```

3. **Verify connections**
   You should see:
   ```
   Connected to gRPC services:
     • GAS (Auth Service): localhost:50051
     • GEI (Schedule Indexer): localhost:50052
   ```

## Breaking Changes

### Environment Variables
- ❌ `GRPC_SERVICE_URL` is **no longer supported**
- ✅ Use `GAS_SERVICE_URL` and `GEI_SERVICE_URL` instead

### Code Changes
If you've extended the server code:

| Old Code | New Code |
|----------|----------|
| `s.grpc.client` | `s.grpcClients.GASClient` |
| `s.grpc.conn` | `s.grpcClients.GASConn` |
| `server.NewGRPCClient(url)` | `server.NewGRPCClients(configs)` |
| `grpcClient.Close()` | `grpcClients.Close()` |

## Rollback Plan

If you need to rollback to the old version:

1. **Git rollback**
   ```bash
   git revert <commit-hash>
   ```

2. **Or manually revert environment variables**
   ```bash
   # Remove new variables
   unset GAS_SERVICE_URL
   unset GEI_SERVICE_URL
   
   # Add old variable
   export GRPC_SERVICE_URL=localhost:50051
   ```

## Common Issues

### Issue: Application won't start
**Error**: `GAS_SERVICE_URL environment variable is required`

**Solution**: 
```bash
# Make sure both variables are set
echo "GAS_SERVICE_URL=localhost:50051" >> .env
echo "GEI_SERVICE_URL=localhost:50052" >> .env
```

### Issue: Connection refused
**Error**: `failed to connect to GEI gRPC service at localhost:50052: connection refused`

**Solution**:
- Verify the GEI service is running
- Check the port number is correct
- Test connection: `telnet localhost 50052`

### Issue: Old environment variable ignored
**Symptom**: Setting `GRPC_SERVICE_URL` has no effect

**Explanation**: This is expected behavior. The old variable is no longer read by the application. Use the new variables instead.

## Benefits of This Change

1. **Scalability**: Easily add more gRPC services without restructuring code
2. **Clarity**: Each service has a clear name and purpose
3. **Maintainability**: Service-specific configurations are isolated
4. **Flexibility**: Services can be deployed independently
5. **Error Handling**: Better error messages indicate which service failed

## Adding More Services

To add a third service (e.g., GXX):

1. Add environment variable: `GXX_SERVICE_URL=localhost:50053`
2. Add to `grpcConfigs` in `main.go`
3. Add fields to `GRPCClients` struct
4. Add case in `NewGRPCClients()` switch statement
5. Add cleanup in `Close()` method

See `GRPC_SERVICES.md` for detailed instructions.

## Support

If you encounter issues during migration:
1. Check the logs for specific error messages
2. Verify all environment variables are set correctly
3. Ensure all gRPC services are running and accessible
4. Review `GRPC_SERVICES.md` for configuration details

## Changelog

### Version 2.1.0 - Multi-Service Support

**Added**:
- Support for multiple gRPC services
- `GRPCClients` struct to manage multiple connections
- `GRPCServiceConfig` for service configuration
- Environment variables: `GAS_SERVICE_URL`, `GEI_SERVICE_URL`
- Documentation: `GRPC_SERVICES.md`
- Migration guide: `MIGRATION_GUIDE.md`

**Changed**:
- Renamed `GRPCClient` to `GRPCClients`
- Updated `NewServer()` to accept `GRPCClients` instead of `GRPCClient`
- Updated all handler references from `s.grpc.client` to `s.grpcClients.GASClient`

**Deprecated**:
- `GRPC_SERVICE_URL` environment variable (use `GAS_SERVICE_URL` instead)
- Single `GRPCClient` struct

**Removed**:
- `NewGRPCClient()` function (replaced by `NewGRPCClients()`)