# Deployment Guide for Logan's 3D Creations

This project supports deployment to both staging and production environments on a DigitalOcean droplet.

## Environments

- **Staging**: <https://logans3dcreations.digitaldrywood.com> (port 8006)
- **Production**: <https://www.logans3dcreations.com> (port 8007)

## Initial Server Setup

### Prerequisites

- SSH access to jarvis.digitaldrywood.com as `apprunner` user
- Go installed on the server
- Nginx installed and configured
- Certbot for SSL certificates
- systemd for service management

### Staging Setup

1. SSH into the server: `make ssh`
2. Run the staging setup script:

   ```bash
   cd /home/apprunner/sites/logans3d-staging
   ./scripts/deploy-setup-staging.sh
   ```

### Production Setup

1. SSH into the server: `make ssh`
2. Run the production setup script:

   ```bash
   cd /home/apprunner/sites/logans3d
   ./scripts/deploy-setup-production.sh
   ```

## Deployment Commands

### Deploy to Staging

```bash
make deploy-staging
```

or simply:

```bash
make deploy
```

### Deploy to Production

```bash
make deploy-production
```

**Note**: Production deployment requires confirmation to prevent accidental deployments.

## Monitoring and Logs

### View Logs

- Staging logs: `make log-staging`
- Production logs: `make log-production`
- Staging web logs: `make log-web-staging`
- Production web logs: `make log-web-production`

### SSH Access

```bash
make ssh
```

## Configuration Files

All server configuration files are stored in the `deployed-configs/` directory:

```text
deployed-configs/
├── staging/
│   ├── systemd/          # systemd service file
│   ├── nginx/            # Nginx site configuration
│   ├── logrotate/        # Log rotation configuration
│   └── environment       # Environment variables
└── production/
    ├── systemd/          # systemd service file
    ├── nginx/            # Nginx site configuration
    ├── logrotate/        # Log rotation configuration
    └── environment       # Environment variables
```

## Manual Service Management

If you need to manually manage the services:

### Staging

```bash
# Start/stop/restart
sudo systemctl start logans3d-staging
sudo systemctl stop logans3d-staging
sudo systemctl restart logans3d-staging

# Check status
sudo systemctl status logans3d-staging
```

### Production

```bash
# Start/stop/restart
sudo systemctl start logans3d
sudo systemctl stop logans3d
sudo systemctl restart logans3d

# Check status
sudo systemctl status logans3d
```

## Database Management

Each environment has its own SQLite database:

- Staging: `/home/apprunner/sites/logans3d-staging/data/database.db`
- Production: `/home/apprunner/sites/logans3d/data/database.db`

To run migrations on the server:

```bash
cd /home/apprunner/sites/logans3d-staging  # or logans3d for production
goose -dir storage/migrations sqlite3 ./data/database.db up
```

## SSL Certificates

SSL certificates are managed by Let's Encrypt and should auto-renew. To manually renew:

```bash
sudo certbot renew
```

## Troubleshooting

1. **Service won't start**: Check logs with `journalctl -u logans3d-staging -n 50`
2. **Port already in use**: Ensure no other service is using ports 8006 (staging) or 8007 (production)
3. **Database errors**: Check database file permissions and ensure the data directory exists
4. **Nginx errors**: Test configuration with `sudo nginx -t` before reloading
