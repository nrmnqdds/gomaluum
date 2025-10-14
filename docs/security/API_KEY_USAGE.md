# API Key Authentication - Enhanced Security Layer

This document explains how to use the new API key authentication system that adds an additional layer of security to your PASETO tokens.

## Overview

The API key system provides an extra encryption layer on top of the existing PASETO token authentication. Before the token data is encrypted with PASETO, it's first encrypted using your API key. This means:

1. **Double encryption**: Data is encrypted with your API key, then with PASETO
2. **Key-specific tokens**: Tokens can only be decrypted with the same API key used during generation
3. **Optional usage**: If no API key is provided, a default key is used for backward compatibility

## Getting Started

### 1. Generate an API Key

First, generate a new API key:

```bash
curl -X POST http://localhost:8080/api/key/generate
```

Response:
```json
{
  "api_key": "gml_1703123456_a1b2c3d4e5f6789012345678901234567890abcd",
  "message": "API key generated successfully. Include this key in the 'x-gomaluum-key' header for enhanced security.",
  "created_at": "now"
}
```

### 2. Login with API Key

Include your API key in the `x-gomaluum-key` header when logging in:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -H "x-gomaluum-key: gml_1703123456_a1b2c3d4e5f6789012345678901234567890abcd" \
  -d '{
    "username": "your_username",
    "password": "your_password"
  }'
```

Response:
```json
{
  "message": "Login successful! Please use the token in the Authorization header for future requests.",
  "data": {
    "token": "v4.public.eyJhdWQiOiJnb21hbHV1bSIs...",
    "username": "your_username"
  }
}
```

### 3. Use Token with API Key

For all subsequent API calls, include both the Bearer token and your API key:

```bash
curl -X GET http://localhost:8080/api/profile \
  -H "Authorization: Bearer v4.public.eyJhdWQiOiJnb21hbHV1bSIs..." \
  -H "x-gomaluum-key: gml_1703123456_a1b2c3d4e5f6789012345678901234567890abcd"
```

## Security Considerations

### API Key Format

API keys come in two formats:
- **Timestamped keys** (recommended): `gml_1703123456_a1b2c3d4e5f6...`
- **Hex keys**: `a1b2c3d4e5f6789012345678901234567890abcd...`

### Key Storage

- **Store securely**: Never commit API keys to version control
- **Environment variables**: Store in environment variables or secure vaults
- **Rotation**: Generate new keys periodically for enhanced security

### Default Key Fallback

If no `x-gomaluum-key` header is provided:
- The system uses a default key: `gomaluum-default-key-2024`
- This maintains backward compatibility with existing implementations
- **Recommendation**: Always use your own generated API key for production

## Implementation Details

### Encryption Flow

1. **Login Request**:
   ```
   User Data → API Key Encryption → PASETO Encryption → Token
   ```

2. **Token Validation**:
   ```
   Token → PASETO Decryption → API Key Decryption → User Data
   ```

### Error Handling

Common error responses:

```json
{
  "message": "Invalid API key",
  "status_code": 401
}
```

```json
{
  "message": "Failed to decrypt data with API key",
  "status_code": 500
}
```

## Migration Guide

### For Existing Applications

1. **No immediate changes required**: Existing tokens continue to work with the default key
2. **Gradual migration**: Start including API keys in new login requests
3. **Full migration**: Eventually require API keys for all users

### Example Migration Steps

1. Generate API key for your application
2. Update your login flow to include the `x-gomaluum-key` header
3. Store the API key securely in your application configuration
4. Include the API key in all API requests

## Best Practices

1. **One key per application**: Use different API keys for different applications/environments
2. **Key rotation**: Regularly generate new API keys and update your applications
3. **Monitoring**: Log API key usage (but never log the actual key values)
4. **Secure transmission**: Always use HTTPS when transmitting API keys
5. **Access control**: Limit who has access to API keys in your organization

## Troubleshooting

### Common Issues

1. **401 Unauthorized**: Check that your API key is included in the `x-gomaluum-key` header
2. **Invalid API key format**: Ensure your API key matches the expected format
3. **Token decryption failed**: Make sure you're using the same API key that was used during login

### Debug Tips

- Check server logs for detailed error messages
- Verify API key format using the validation endpoint
- Test with the default key first, then switch to your generated key

## Examples

### Node.js/JavaScript
```javascript
const apiKey = 'gml_1703123456_a1b2c3d4e5f6789012345678901234567890abcd';

// Login
const loginResponse = await fetch('/api/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'x-gomaluum-key': apiKey
  },
  body: JSON.stringify({
    username: 'your_username',
    password: 'your_password'
  })
});

// Use API
const profileResponse = await fetch('/api/profile', {
  headers: {
    'Authorization': `Bearer ${token}`,
    'x-gomaluum-key': apiKey
  }
});
```

### Python
```python
import requests

api_key = 'gml_1703123456_a1b2c3d4e5f6789012345678901234567890abcd'
headers = {'x-gomaluum-key': api_key}

# Login
login_response = requests.post('/api/auth/login', 
  headers={**headers, 'Content-Type': 'application/json'},
  json={'username': 'your_username', 'password': 'your_password'}
)

# Use API
token = login_response.json()['data']['token']
profile_response = requests.get('/api/profile', 
  headers={**headers, 'Authorization': f'Bearer {token}'}
)
```

### Go
```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

func main() {
    apiKey := "gml_1703123456_a1b2c3d4e5f6789012345678901234567890abcd"
    
    // Login
    loginData := map[string]string{
        "username": "your_username",
        "password": "your_password",
    }
    jsonData, _ := json.Marshal(loginData)
    
    req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("x-gomaluum-key", apiKey)
    
    // Use API
    req2, _ := http.NewRequest("GET", "/api/profile", nil)
    req2.Header.Set("Authorization", "Bearer "+token)
    req2.Header.Set("x-gomaluum-key", apiKey)
}
```
