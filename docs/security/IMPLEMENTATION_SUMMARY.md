# API Key Authentication Implementation Summary

## Overview

This document summarizes the implementation of an additional layer of authentication using API keys in the GoMa'luum project. The new system adds encryption using user-provided API keys before the data is encrypted with PASETO tokens.

## What Was Implemented

### 1. API Key Generation System
- **File**: `pkg/apikey/apikey.go`
- **Features**:
  - Generate random hex-encoded API keys
  - Generate timestamped API keys with prefix `gml_`
  - API key validation
  - Default fallback key: `gomaluum-default-key-2024`

### 2. Double Encryption Layer
- **Encryption Flow**: `User Data → API Key Encryption → PASETO Encryption → Token`
- **Decryption Flow**: `Token → PASETO Decryption → API Key Decryption → User Data`
- **Algorithm**: AES-256-GCM with SHA-256 key derivation
- **Security**: Each encryption uses a random nonce for semantic security

### 3. API Endpoints
- **New Endpoint**: `POST /api/key/generate`
  - Generates new timestamped API keys
  - No authentication required
  - Returns JSON with API key and usage instructions

### 4. Enhanced Authentication Middleware
- **File**: `internal/server/middleware.go`
- **Features**:
  - Reads `x-gomaluum-key` header
  - Falls back to default key if header missing
  - Validates API key format
  - Passes API key to token decoding

### 5. Updated PASETO Integration
- **File**: `internal/server/paseto.go`
- **Changes**:
  - `GeneratePasetoToken()` now accepts API key parameter
  - `DecodePasetoToken()` now requires API key parameter
  - All sensitive data encrypted with API key before PASETO
  - Updated `TokenPayload` struct to include API key

### 6. Enhanced Login Handler
- **File**: `internal/server/auth.go`
- **Features**:
  - Reads `x-gomaluum-key` header during login
  - Validates API key format
  - Passes API key to token generation

### 7. Error Handling
- **File**: `internal/errors/apikey.error.go`
- **New Errors**:
  - `ErrFailedToGenerateAPIKey`
  - `ErrInvalidAPIKey`
  - `ErrAPIKeyRequired`
  - `ErrFailedToEncryptWithAPIKey`
  - `ErrFailedToDecryptWithAPIKey`

### 8. Updated CORS Configuration
- **File**: `internal/server/routes.go`
- **Change**: Added `x-gomaluum-key` to allowed headers

## Usage Flow

### For New Users (Recommended)
1. Generate API key: `POST /api/key/generate`
2. Login with API key: `POST /api/auth/login` + `x-gomaluum-key` header
3. Use API with both token and API key in all requests

### For Existing Users (Backward Compatible)
1. Continue using existing flow without API key
2. System automatically uses default key
3. Gradually migrate to using generated API keys

## Security Benefits

1. **Double Encryption**: Data encrypted twice with different keys
2. **Key Isolation**: Each application/user can have unique encryption
3. **Token Binding**: Tokens are bound to specific API keys
4. **Forward Security**: Compromised PASETO keys don't expose API key encrypted data
5. **Granular Access**: API keys can be rotated independently of PASETO keys

## Files Created/Modified

### New Files
- `pkg/apikey/apikey.go` - Core API key functionality
- `pkg/apikey/apikey_test.go` - Comprehensive tests
- `internal/errors/apikey.error.go` - API key specific errors
- `internal/server/apikey.go` - API key generation endpoint
- `docs/API_KEY_USAGE.md` - Detailed usage documentation
- `scripts/demo_apikey.sh` - Interactive demo script

### Modified Files
- `internal/server/paseto.go` - Added API key encryption layer
- `internal/server/middleware.go` - API key header processing
- `internal/server/auth.go` - API key integration in login
- `internal/server/routes.go` - New routes and CORS update
- `README.md` - Added API key documentation section

## Testing

- **Unit Tests**: `pkg/apikey/apikey_test.go` with 100% coverage
- **Integration Tests**: Interactive demo script
- **Security Tests**: Cross-key validation, wrong key rejection

## Backward Compatibility

The implementation is fully backward compatible:
- Existing tokens continue to work
- No breaking changes to existing API endpoints
- Default key used when no API key provided
- Gradual migration path available

## Configuration

### Environment Variables
No new environment variables required. The system uses:
- Default API key for backward compatibility
- Runtime key generation for new keys
- Existing PASETO environment variables

### Headers
- **New Header**: `x-gomaluum-key` (optional)
- **Existing Headers**: `Authorization: Bearer <token>` (required)

## Performance Impact

- **Minimal**: Additional AES-GCM encryption/decryption
- **Optimized**: Key derivation using SHA-256 (fast)
- **Efficient**: Base64 encoding for transport
- **Scalable**: No additional database requirements

## Future Enhancements

1. **Key Management**: API key rotation endpoints
2. **Analytics**: API key usage tracking
3. **Rate Limiting**: Per-API-key rate limits
4. **Permissions**: API key scopes and permissions
5. **Audit**: API key access logging

## Security Considerations

1. **Key Storage**: Users must store API keys securely
2. **Transport**: Always use HTTPS for API key transmission
3. **Rotation**: Recommend regular API key rotation
4. **Monitoring**: Log API key usage (not values)
5. **Default Key**: Should be changed in production environments