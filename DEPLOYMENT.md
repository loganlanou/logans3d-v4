# Deployment Guide for Logan's 3D Creations

This project is deployed to a single production environment on a self-hosted VPS.

## Production Environment

- **URL**: https://www.logans3dcreations.com
- **Server**: jarvis.digitaldrywood.com (DigitalOcean VPS)
- **Port**: 8007
- **User**: apprunner
- **Service**: logans3d (systemd)

## Important: Environment Variables Are NOT in Git

**CRITICAL**: The `/etc/logans3d/environment` file is:
- **ONLY on the production server** (not in git)
- **Manually created and edited** on the server
- **Contains real secrets** (API keys, SMTP credentials, etc.)
- **Never, ever committed to git**

This design ensures secrets are never exposed in version control.

## Makefile Commands

### SSH to Server
```bash
make ssh
# Opens interactive SSH session to apprunner@jarvis.digitaldrywood.com
```

### Deploy Code Changes
```bash
make deploy
# or: make deploy-production
```

This command:
1. Confirms you want to deploy to production
2. SSHs to the server
3. Pulls latest code from git
4. Runs code generation (`go generate ./...`)
5. Builds the binary
6. Restarts the service

**Note**: Code deployment does NOT modify environment variables.

### View Production Logs
```bash
make log-production
# View systemd logs (journalctl) - follow mode

make log-web-production
# View application logs file
```

### Manage Environment Variables
```bash
# View all production environment variables
make env-view

# Set a single environment variable (restarts service)
make env-set KEY=VALUE

# Examples:
make env-set EMAIL_FROM=new-email@example.com
make env-set BREVO_SMTP_KEY=new-api-key-here
```

After setting environment variables, the service automatically restarts to load them.

## Initial Server Setup (One-Time)

This section describes the one-time setup required when first provisioning the server. This has already been done for production.

### Prerequisites

Ensure the server has:
- Go 1.25+
- Node.js 18+ (for CSS compilation)
- Nginx (reverse proxy)
- Certbot (SSL certificates)
- systemd (service management)

### Step 1: Run Setup Script

```bash
make ssh

# From the cloned repository directory on the server:
cd /home/apprunner/sites/logans3d
./scripts/deploy-setup-production.sh
```

The setup script will:
- Create necessary directories
- Clone/update the git repository
- Install systemd service and nginx configuration
- Build the application
- Create the environment file

### Step 2: Manually Configure Environment Variables

**CRITICAL**: After the setup script completes, you MUST manually create the environment file with real secrets.

```bash
# SSH to server
make ssh

# Edit the environment file
sudo nano /etc/logans3d/environment
```

The environment file should contain all required configuration. Here's a template:

```bash
# Logan's 3D Production Environment Variables
# NEVER commit this file to git
# Keep this file secure (chmod 600)

# Application Settings
PORT=8007
ENVIRONMENT=production
BASE_URL=https://www.logans3dcreations.com

# Database
DB_PATH=/home/apprunner/sites/logans3d/data/database.db

# Logging
LOG_LEVEL=info
LOG_FILE_PATH=/var/log/logans3d/logans3d.log

# Clerk Authentication (Production Keys)
CLERK_PUBLISHABLE_KEY=pk_live_YOUR_ACTUAL_KEY
CLERK_SECRET_KEY=sk_live_YOUR_ACTUAL_KEY

# JWT Security (generate with: openssl rand -base64 32)
JWT_SECRET=YOUR_ACTUAL_JWT_SECRET

# Stripe Payment Processing (Production Keys)
STRIPE_PUBLISHABLE_KEY=pk_live_YOUR_ACTUAL_KEY
STRIPE_SECRET_KEY=sk_live_YOUR_ACTUAL_KEY
STRIPE_WEBHOOK_SECRET=whsec_YOUR_ACTUAL_WEBHOOK_SECRET

# Email (Brevo SMTP)
EMAIL_FROM=prints@logans3dcreations.com
EMAIL_TO_INTERNAL=prints@logans3dcreations.com
BREVO_SMTP_HOST=smtp-relay.brevo.com
BREVO_SMTP_PORT=587
BREVO_SMTP_LOGIN=YOUR_BREVO_LOGIN
BREVO_SMTP_KEY=YOUR_BREVO_SMTP_KEY

# Google reCAPTCHA v3
RECAPTCHA_SITE_KEY=YOUR_PRODUCTION_SITE_KEY
RECAPTCHA_SECRET_KEY=YOUR_PRODUCTION_SECRET_KEY
RECAPTCHA_MIN_SCORE=0.5

# EasyPost API (Shipping)
EASYPOST_API_KEY=EZPK_YOUR_PRODUCTION_KEY

# File Uploads
UPLOAD_MAX_SIZE=104857600
UPLOAD_DIR=/home/apprunner/sites/logans3d/public/uploads

# Backup with Litestream (optional)
LITESTREAM_ACCESS_KEY_ID=YOUR_AWS_ACCESS_KEY
LITESTREAM_SECRET_ACCESS_KEY=YOUR_AWS_SECRET_KEY
LITESTREAM_BUCKET=your-s3-bucket-name
```

**Save the file** (Ctrl+X, Y, Enter), then set proper permissions:

```bash
sudo chmod 600 /etc/logans3d/environment
sudo chown root:root /etc/logans3d/environment

# Verify permissions
ls -la /etc/logans3d/environment
# Should show: -rw------- 1 root root
```

### Step 3: Start the Service

```bash
# Reload systemd and start the service
sudo systemctl daemon-reload
sudo systemctl enable logans3d
sudo systemctl start logans3d

# Verify it's running
sudo systemctl status logans3d

# Follow logs to check for startup errors
sudo journalctl -u logans3d -f
```

## Regular Deployment Workflow

For deploying code changes:

```bash
# 1. Commit and push your changes locally
git add .
git commit -m "your commit message"
git push

# 2. Deploy to production (confirms before proceeding)
make deploy

# 3. Monitor logs for any errors
make log-production
```

## Updating Environment Variables

To update environment variables on production:

**Single variable:**
```bash
make env-set NEW_VAR=new_value
```

**Multiple variables:**
```bash
make env-set VAR1=value1
make env-set VAR2=value2
make env-set VAR3=value3
```

**Manual edit:**
```bash
make ssh
sudo nano /etc/logans3d/environment
# Save and exit, service auto-restarts
```

## Manual Service Management

If you need to directly manage the service:

```bash
# Check status
sudo systemctl status logans3d

# Start/stop/restart
sudo systemctl start logans3d
sudo systemctl stop logans3d
sudo systemctl restart logans3d

# View logs
sudo journalctl -u logans3d -f              # Follow logs
sudo journalctl -u logans3d -n 100          # View last 100 lines
sudo journalctl -u logans3d --since "1 hour ago"  # View last hour

# View application logs file
sudo tail -f /var/log/logans3d/logans3d.log
```

## Database Management

Database file location: `/home/apprunner/sites/logans3d/data/database.db`

To run migrations on the server:

```bash
make ssh
cd /home/apprunner/sites/logans3d
goose -dir storage/migrations sqlite3 ./data/database.db up
```

## SSL Certificates

SSL certificates are managed by Let's Encrypt and should auto-renew. To manually renew:

```bash
make ssh
sudo certbot renew
```

## Troubleshooting

### Service Won't Start

Check the logs:
```bash
sudo journalctl -u logans3d -n 50
make log-production
```

Common causes:
- Missing or invalid environment variables
- Port 8007 already in use
- Database connection issues
- Missing configuration files

### Port Already in Use

Check what's using port 8007:
```bash
sudo lsof -i :8007
sudo ss -tlnp | grep 8007
```

Kill the process if needed and restart the service:
```bash
sudo kill -9 <PID>
sudo systemctl restart logans3d
```

### Database Errors

Check database file and directory permissions:
```bash
ls -la /home/apprunner/sites/logans3d/data/
```

The directory should be owned by `apprunner:apprunner`:
```bash
sudo chown -R apprunner:apprunner /home/apprunner/sites/logans3d/data
```

### Nginx Errors

Test nginx configuration:
```bash
sudo nginx -t
```

Reload nginx if configuration is valid:
```bash
sudo systemctl reload nginx
```

## Security Considerations

1. **Environment Files**: Never commit environment files to git
2. **File Permissions**: Environment file should be `600` (readable by root only)
3. **SSH Keys**: Use SSH keys for authentication, not passwords
4. **Secrets**: Store all secrets in the environment file, never in code
5. **Backups**: Regular backups of the database are essential (use Litestream if configured)

## Support

For issues or questions:
1. Check the logs with `make log-production` or `make log-web-production`
2. SSH to the server with `make ssh`
3. Review this deployment guide
