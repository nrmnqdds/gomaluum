<img width="253" height="43" alt="gomaluum-v1" src="https://raw.githubusercontent.com/nrmnqdds/gomaluum/refs/heads/main/assets/gomaluum-logo-v2.png" />

## 🚧 **In Construction** 🚧

> [!IMPORTANT]
> This project is **not** associated with the official i-Ma'luum!

A proxy API which enables developers to build applications on top of i-Ma'luum.
Primarily used by some IIUM's student-made app:

- [Simplified i-Ma'luum](https://imaluum.quddus.my)
- [ProReg](https://proreg.app)

Swagger API documentation is available at [here](https://api.quddus.my/api/reference).

## 🔐 Enhanced Security with API Keys

GoMa'luum now supports an additional layer of authentication using API keys. This provides enhanced security by encrypting your data with a unique key before PASETO token encryption.

### Quick Start with API Keys

1. **Generate an API Key**:

   ```bash
   curl -X POST https://api.quddus.my/api/key/generate
   ```

2. **Login with your API Key**:

   ```bash
   curl -X POST https://api.quddus.my/api/auth/login \
     -H "Content-Type: application/json" \
     -H "x-gomaluum-key: YOUR_API_KEY" \
     -d '{"username": "your_username", "password": "your_password"}'
   ```

3. **Use the API Key in all requests**:
   ```bash
   curl -X GET https://api.quddus.my/api/profile \
     -H "Authorization: Bearer YOUR_TOKEN" \
     -H "x-gomaluum-key: YOUR_API_KEY"
   ```

### Security Benefits

- **Double Encryption**: Data is encrypted with your API key, then with PASETO
- **Key-Specific Access**: Tokens can only be used with the same API key
- **Backward Compatible**: Works without API keys using a default key
- **Enhanced Privacy**: Each application can have its own unique encryption layer

For detailed documentation, see [API Key Usage Guide](docs/API_KEY_USAGE.md).

## How it works under the hood

```mermaid
flowchart TD
    A["User sends request to GoMa'luum API"] --> B["GoMa'luum receives request"]
    B --> C["GoMa'luum sends request to original endpoint (imaluum.edu.my)"]
    C --> D["imaluum.edu.my returns auth cookie"]
    D --> E["GoMa'luum stores auth cookie, wraps with PASETO token"]
    E --> F["GoMa'luum uses cookie to scrape user data from imaluum.edu.my"]
    F --> G["GoMa'luum processes and formats scraped data"]
    G --> H["GoMa'luum returns pretty JSON/API response to user"]
    H --> I["User views data in a pretty UI or via API"]
    E --> J["PASETO token used for secure SSO/session management"]
    B --> K["Request validation, logging, and error handling"]
    F --> L["Concurrent scraping using goroutines for performance"]
    G --> M["Optional: Cache data for faster repeated access"]
    B --> N["gRPC support for internal/external service communication"]
```

## Local installation

### Prerequisites

- Go 1.23 or higher
- PostgreSQL database (optional, for analytics functionality)

### Setup

```bash
git clone http://github.com/nrmnqdds/gomaluum
cd gomaluum
go mod tidy

# Copy .env.example to .env and configure
cp .env.example .env
# Edit .env and set GRPC_SERVICE_URL to your external gRPC service
# Example: GRPC_SERVICE_URL=localhost:50051 or your-service.com:443

# Run the application
air
```

For detailed database setup instructions (if you want analytics functionality), see [Database Configuration](docs/DATABASE.md).

## Using Docker

```bash
docker build -t gomaluum .
# Without database (no analytics)
docker run -p 1323:1323 -e GRPC_SERVICE_URL=your-grpc-service:50051 -d gomaluum

# With database (analytics enabled)
docker run -p 1323:1323 \
  -e GRPC_SERVICE_URL=your-grpc-service:50051 \
  -e DATABASE_URL=postgresql://user:pass@host:5432/db \
  -d gomaluum
```

> **Note**: The `GRPC_SERVICE_URL` environment variable is required and must point to your external gRPC authentication service.

For a complete Docker Compose setup with PostgreSQL, see [Database Configuration](docs/DATABASE.md).

## Support this project!

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/nrmnqdds)
