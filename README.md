# O Dan Go - Setup Instructions

## Environment Setup

### Prerequisites
- Go 1.19 or later
- Git
- Access to NetSapiens API credentials

### Initial Setup

1. **Clone and navigate to the project:**
   ```bash
   git clone <your-repo-url>
   cd o-dan-go
   ```

2. **Install dependencies:**
   ```bash
   go mod tidy
   ```

3. **Copy the environment template:**
   ```bash
   cp .env.example .env
   ```

4. **Edit `.env` with your actual credentials:**
   ```bash
   nano .env
   ```
   
   Update the following values:
   ```
   NETSAPIENS_BASE_URL=https://your-netsapiens-api-url
   NETSAPIENS_ACCESS_TOKEN=your_actual_oauth_token_here
   NETSAPIENS_CLIENT_ID=your_client_id_here
   NETSAPIENS_CLIENT_SECRET=your_client_secret_here
   ```

## Running the Application

### Development Mode

1. **Start the web server:**
   ```bash
   go run main.go
   ```
   
   The server will start on port 8080 (or the port specified in your `.env` file).

2. **Access the application:**
   - Web interface: http://localhost:8080
   - Health check: http://localhost:8080/ (should return JSON with status)

### Testing CDR Discovery

1. **Test CDR endpoint connectivity:**
   ```bash
   go run main.go test-cdr
   ```
   
   This will:
   - Initialize the CDR Discovery Service
   - Test all configured endpoints
   - Show detailed results and timing
   - Display sample CDR data if available

### Building for Production

1. **Build the executable:**
   ```bash
   go build -o odango-app
   ```

2. **Run the built application:**
   ```bash
   ./odango-app
   ```

## Production Deployment

### Environment Variables (Recommended)

For production servers, set environment variables directly instead of using a `.env` file:

```bash
export NETSAPIENS_ACCESS_TOKEN="your_production_token"
export NETSAPIENS_BASE_URL="https://prod-api.netsapiens.com"
export APP_ENV="production"
export APP_PORT="8080"
export DATABASE_PATH="/var/lib/odango/odango.db"
```

### Ubuntu Server Deployment

1. **Create application directory:**
   ```bash
   sudo mkdir -p /opt/odango
   sudo chown $USER:$USER /opt/odango
   ```

2. **Copy application:**
   ```bash
   cp odango-app /opt/odango/
   cp -r static /opt/odango/ # If you have static files
   ```

3. **Create systemd service** (`/etc/systemd/system/odango.service`):
   ```ini
   [Unit]
   Description=O Dan Go NetSapiens API Service
   After=network.target

   [Service]
   Type=simple
   User=odango
   WorkingDirectory=/opt/odango
   ExecStart=/opt/odango/odango-app
   Restart=always
   RestartSec=10

   # Environment variables
   Environment=APP_ENV=production
   Environment=APP_PORT=8080
   Environment=NETSAPIENS_BASE_URL=https://your-api-url
   Environment=NETSAPIENS_ACCESS_TOKEN=your_token

   [Install]
   WantedBy=multi-user.target
   ```

4. **Start and enable service:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable odango
   sudo systemctl start odango
   sudo systemctl status odango
   ```

## Configuration Options

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `NETSAPIENS_BASE_URL` | NetSapiens API base URL | `https://ns-api.com` | No |
| `NETSAPIENS_ACCESS_TOKEN` | OAuth access token | - | **Yes** |
| `NETSAPIENS_CLIENT_ID` | OAuth client ID | - | No* |
| `NETSAPIENS_CLIENT_SECRET` | OAuth client secret | - | No* |
| `APP_ENV` | Environment (development/production) | `development` | No |
| `APP_PORT` | Server port | `8080` | No |
| `DATABASE_PATH` | SQLite database file path | `./data/odango.db` | No |

*Required for OAuth flow implementation

### Development vs Production

**Development Mode:**
- Loads `.env` file automatically
- Gin runs in debug mode
- Detailed error messages
- Hot reload with tools like `air`

**Production Mode:**
- Uses system environment variables
- Gin runs in release mode
- Structured logging
- Performance optimizations

## Troubleshooting

### Common Issues

1. **"NETSAPIENS_ACCESS_TOKEN is required but not set"**
   - Ensure your `.env` file exists and contains the token
   - Check that the token is not empty or commented out

2. **"connection refused" errors during testing**
   - Verify your `NETSAPIENS_BASE_URL` is correct
   - Check that your access token is valid and not expired
   - Ensure network connectivity to the NetSapiens API

3. **"no such file or directory" when running built binary**
   - Make sure you built for the correct architecture
   - For Ubuntu server: `GOOS=linux GOARCH=amd64 go build -o odango-app`

### Debug Mode

Enable verbose logging by setting:
```bash
export GIN_MODE=debug
export LOG_LEVEL=debug
```

### API Testing

Test individual API endpoints manually:
```bash
curl -H "Authorization: Bearer $NETSAPIENS_ACCESS_TOKEN" \
     -H "Accept: application/json" \
     "$NETSAPIENS_BASE_URL/cdrs?limit=1"
```

## Development Workflow

### Adding New Features

1. Create feature branch:
   ```bash
   git checkout -b feature/new-feature-name
   ```

2. Make changes and test:
   ```bash
   go run main.go test-cdr
   go test ./...
   ```

3. Commit and push:
   ```bash
   git add .
   git commit -m "Add new feature description"
   git push origin feature/new-feature-name
   ```

### Database Management

The application uses SQLite for data storage:

- **Development**: Database stored in `./data/odango.db`
- **Production**: Configurable via `DATABASE_PATH` environment variable
- **Migrations**: Handled automatically on startup

## Security Considerations

- Never commit `.env` files to version control
- Use strong, unique API tokens
- Rotate credentials regularly
- Run production servers with limited user privileges
- Use HTTPS in production deployments
- Regularly update dependencies: `go get -u && go mod tidy`

## Getting Help

1. Check application logs:
   ```bash
   # Development
   go run main.go
   
   # Production (systemd)
   sudo journalctl -u odango -f
   ```

2. Test connectivity:
   ```bash
   go run main.go test-cdr
   ```

3. Verify configuration:
   ```bash
   # Check environment variables are loaded
   go run -ldflags="-X main.showConfig=true" main.go
   ```