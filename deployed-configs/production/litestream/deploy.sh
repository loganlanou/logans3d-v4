#!/bin/bash
# Litestream Deployment Script for Production
# Run this on the production server as root or with sudo

set -e

echo "=== Litestream v0.3.13 Deployment Script ==="
echo ""

# Check if running as root or with sudo
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: Please run as root or with sudo"
    exit 1
fi

echo "Step 1: Pull latest configs from git..."
cd /home/apprunner/sites/logans3d
sudo -u apprunner git pull origin main

echo ""
echo "Step 2: Download and install Litestream v0.3.13..."
cd /tmp
wget -q https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.deb
dpkg -i litestream-v0.3.13-linux-amd64.deb
litestream version

echo ""
echo "Step 3: Create required directories..."
mkdir -p /etc/litestream
mkdir -p /var/log/litestream
chown -R apprunner:apprunner /var/log/litestream

echo ""
echo "Step 4: Deploy Litestream configuration..."
cp /home/apprunner/sites/logans3d/deployed-configs/production/litestream/litestream.yml /etc/litestream/litestream.yml
chown root:root /etc/litestream/litestream.yml
chmod 644 /etc/litestream/litestream.yml

echo ""
echo "Step 5: Create environment file with credentials..."
echo "IMPORTANT: You need to manually create /etc/litestream/environment"
echo ""
echo "Run this command and paste your credentials:"
echo "  sudo nano /etc/litestream/environment"
echo ""
echo "Paste this content (with your actual keys):"
echo "---"
cat <<'ENVEOF'
LITESTREAM_ACCESS_KEY_ID=DO8016Q3ZQ68C4K9FJP6
LITESTREAM_SECRET_ACCESS_KEY=KDQJsIEV3/FwspYTIf1sEM23+Cd1sx1qAwU1Iw9Yy6s
DIGITALOCEAN_SPACES_ENDPOINT=nyc3.digitaloceanspaces.com
DIGITALOCEAN_SPACES_REGION=nyc3
ENVEOF
echo "---"
echo ""
read -p "Press ENTER after you've created /etc/litestream/environment..."

# Verify environment file exists
if [ ! -f /etc/litestream/environment ]; then
    echo "ERROR: /etc/litestream/environment not found"
    exit 1
fi

# Set proper permissions
chown root:root /etc/litestream/environment
chmod 600 /etc/litestream/environment

echo ""
echo "Step 6: Deploy systemd service..."
cp /home/apprunner/sites/logans3d/deployed-configs/production/systemd/litestream.service /etc/systemd/system/litestream.service
chown root:root /etc/systemd/system/litestream.service
chmod 644 /etc/systemd/system/litestream.service

echo ""
echo "Step 7: Enable and start Litestream service..."
systemctl daemon-reload
systemctl enable litestream
systemctl start litestream

echo ""
echo "Step 8: Check service status..."
sleep 2
systemctl status litestream --no-pager

echo ""
echo "Step 9: View logs..."
tail -20 /var/log/litestream/litestream.log

echo ""
echo "=== Deployment Complete! ==="
echo ""
echo "Monitor logs with:"
echo "  sudo tail -f /var/log/litestream/litestream.log"
echo ""
echo "Check backup status with:"
echo "  litestream snapshots -config /etc/litestream/litestream.yml /home/apprunner/sites/logans3d/data/database.db"
