# Litestream v0.3.x Deployment Guide for Production

This guide covers deploying Litestream v0.3.13 as a systemd service on your production DigitalOcean droplet with DigitalOcean Spaces for off-server backups.

## Overview

Litestream provides continuous SQLite database replication to DigitalOcean Spaces. This configuration provides protection against:
- Database corruption
- Accidental deletion
- Application errors that corrupt data
- Complete server failure
- Point-in-time recovery (7 days retention)

## Prerequisites

- SSH access to production server
- Root or sudo access
- SQLite database running at: `/home/apprunner/sites/logans3d/data/database.db`
- DigitalOcean Spaces bucket created (see "DigitalOcean Spaces Setup" below)
- Spaces API credentials (access key and secret key)

## DigitalOcean Spaces Setup

### 1. Create a Spaces Bucket

```bash
# Via DigitalOcean Console:
1. Navigate to Spaces in the DigitalOcean console
2. Click "Create" > "Spaces"
3. Choose your region (e.g., nyc3, sfo3) - ideally the same region as your droplet
4. Name your Space: "digital-drywood"
5. Choose "Restrict File Listing" for security
6. Click "Create Space"
```

**Important**: You can use a single bucket for multiple databases by using different paths in the configuration. For example:
- `digital-drywood` bucket with path `logans3d-production` for this database
- `digital-drywood` bucket with path `other-app-production` for another database
- `digital-drywood` bucket with path `staging-db` for staging databases

This saves money since you only pay $5/month for one Space (250GB + 1TB transfer).

### 2. Generate API Keys

```bash
# In the Spaces UI:
1. Click "Manage Keys" in the left sidebar (or go to API > Spaces Keys)
2. Click "Generate New Key"
3. Name it: "litestream-backup"
4. Save both the Access Key and Secret Key - you'll need these for the environment file
```

## Installation Steps

### 1. Download and Install Litestream v0.3.13

```bash
# SSH into production server
ssh apprunner@your-server-ip

# Download Litestream v0.3.13
wget https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.deb

# Install the package
sudo dpkg -i litestream-v0.3.13-linux-amd64.deb

# Verify installation
litestream version
```

### 2. Create Required Directories

```bash
# Create Litestream config directory
sudo mkdir -p /etc/litestream

# Create log directory
sudo mkdir -p /var/log/litestream

# Set proper ownership
sudo chown -R apprunner:apprunner /var/log/litestream
```

### 3. Deploy Configuration Files

```bash
# Copy Litestream configuration
sudo cp /path/to/deployed-configs/production/litestream/litestream.yml /etc/litestream/litestream.yml

# Copy environment file with Spaces credentials
sudo cp /path/to/deployed-configs/production/litestream/environment /etc/litestream/environment

# IMPORTANT: Edit the environment file with your actual Spaces credentials
sudo nano /etc/litestream/environment
# Update these values:
#   LITESTREAM_ACCESS_KEY_ID=your_actual_access_key
#   LITESTREAM_SECRET_ACCESS_KEY=your_actual_secret_key
#   DIGITALOCEAN_SPACES_ENDPOINT=nyc3.digitaloceanspaces.com  # or your region
#   DIGITALOCEAN_SPACES_REGION=nyc3  # or your region

# Copy systemd service file
sudo cp /path/to/deployed-configs/production/systemd/litestream.service /etc/systemd/system/litestream.service

# Verify permissions
sudo chown root:root /etc/litestream/litestream.yml
sudo chmod 644 /etc/litestream/litestream.yml
sudo chown root:root /etc/litestream/environment
sudo chmod 600 /etc/litestream/environment  # Restrict access to credentials
sudo chown root:root /etc/systemd/system/litestream.service
sudo chmod 644 /etc/systemd/system/litestream.service
```

### 4. Enable and Start Litestream Service

```bash
# Reload systemd to pick up new service
sudo systemctl daemon-reload

# Enable Litestream to start on boot
sudo systemctl enable litestream

# Start Litestream
sudo systemctl start litestream

# Check status
sudo systemctl status litestream

# View logs
sudo tail -f /var/log/litestream/litestream.log
```

## Configuration Details

### Backup Location
- **Database**: `/home/apprunner/sites/logans3d/data/database.db`
- **Remote Backup**: DigitalOcean Spaces bucket `digital-drywood` with path `logans3d-production`
- **Logs**: `/var/log/litestream/litestream.log`

### Backup Settings
- **Retention**: 7 days (168 hours)
- **Sync Interval**: 10 seconds (WAL changes synced every 10s)
- **Snapshot Interval**: 24 hours (full snapshot daily)
- **Estimated Cost**: ~$5/month for the Space (includes 250GB storage + 1TB transfer)

### How It Works
1. Litestream continuously monitors the SQLite WAL (Write-Ahead Log)
2. Every 10 seconds, it syncs WAL changes to DigitalOcean Spaces
3. Every 24 hours, it creates a full snapshot
4. Old backups are automatically removed after 7 days
5. Backups are stored off-server, protecting against complete server failure

## Monitoring and Maintenance

### Check Service Status
```bash
sudo systemctl status litestream
```

### View Live Logs
```bash
sudo tail -f /var/log/litestream/litestream.log
```

### Check Backup Status
```bash
# List all replicas
litestream replicas -config /etc/litestream/litestream.yml

# Show database information
litestream databases -config /etc/litestream/litestream.yml

# View backup snapshots in Spaces
litestream snapshots -config /etc/litestream/litestream.yml /home/apprunner/sites/logans3d/data/database.db
```

### Restart Service
```bash
sudo systemctl restart litestream
```

## Restoring from Backup

### To a Specific Point in Time
```bash
# Stop the application
sudo systemctl stop logans3d

# Restore database to a specific time
litestream restore -config /etc/litestream/litestream.yml \
  -if-replica-exists \
  -timestamp 2025-10-22T15:30:00Z \
  /home/apprunner/sites/logans3d/data/database.db

# Start the application
sudo systemctl start logans3d
```

### To Latest Backup
```bash
# Stop the application
sudo systemctl stop logans3d

# Restore to latest
litestream restore -config /etc/litestream/litestream.yml \
  -if-replica-exists \
  /home/apprunner/sites/logans3d/data/database.db

# Start the application
sudo systemctl start logans3d
```

## Cost and Storage Considerations

### DigitalOcean Spaces Pricing
- **Base cost**: $5/month for 250GB storage + 1TB transfer
- **Additional storage**: $0.02/GB beyond 250GB
- **Additional transfer**: $0.01/GB beyond 1TB

### Typical Backup Size
The backup size in Spaces will typically be:
- Similar to your database size (for snapshots)
- Plus incremental WAL files
- Automatically cleaned up after 7 days retention

### Multiple Databases in One Bucket
To add more databases to the same `digital-drywood` bucket, add them to your config with different paths:

```yaml
dbs:
  - path: /home/apprunner/sites/logans3d/data/database.db
    replicas:
      - type: s3
        bucket: digital-drywood
        path: logans3d-production  # unique path for this database
        # ... rest of config

  - path: /home/apprunner/sites/other-app/data/database.db
    replicas:
      - type: s3
        bucket: digital-drywood
        path: other-app-production  # different path for this database
        # ... rest of config
```

This allows you to back up multiple SQLite databases without paying for additional Spaces.

## Troubleshooting

### Service Won't Start
```bash
# Check service status
sudo systemctl status litestream

# View detailed logs
sudo journalctl -u litestream -n 50 --no-pager

# Check config syntax
litestream replicate -config /etc/litestream/litestream.yml -dry-run
```

### Permission Issues
```bash
# Ensure apprunner owns the log directory
sudo chown -R apprunner:apprunner /var/log/litestream
```

### Database Locked Errors
- Litestream should never lock the database as it only reads the WAL
- If you see lock errors, ensure only one Litestream instance is running
- Check: `ps aux | grep litestream`

### Connection Reset by Peer Errors
- These are common with DigitalOcean Spaces and do NOT affect backup integrity
- Litestream automatically retries failed requests
- No action needed unless backups stop being created

### Authentication Errors
```bash
# Verify environment variables are set correctly
sudo cat /etc/litestream/environment

# Check that the service can read the environment file
sudo systemctl show litestream | grep Environment

# Test connectivity to Spaces
litestream replicate -config /etc/litestream/litestream.yml -dry-run
```

## Testing Your Backup

After deployment, test your backup setup:

```bash
# Wait a few minutes for initial replication
sleep 60

# Create a test table
sqlite3 /home/apprunner/sites/logans3d/data/database.db "CREATE TABLE IF NOT EXISTS backup_test (id INTEGER PRIMARY KEY, created_at TEXT);"
sqlite3 /home/apprunner/sites/logans3d/data/database.db "INSERT INTO backup_test (created_at) VALUES (datetime('now'));"

# Wait for sync (10 seconds configured)
sleep 15

# Verify backup contains the changes
litestream snapshots -config /etc/litestream/litestream.yml /home/apprunner/sites/logans3d/data/database.db

# Clean up test table
sqlite3 /home/apprunner/sites/logans3d/data/database.db "DROP TABLE IF EXISTS backup_test;"
```

## Support and Documentation

- Litestream v0.3 Documentation: https://litestream.io
- GitHub Issues: https://github.com/benbjohnson/litestream/issues
- Note: v0.5.x has known stability issues, so we're using v0.3.13
