#!/bin/bash
# Setup script for Logan's 3D staging environment

echo "🚀 Setting up Logan's 3D Staging Environment..."

# Create necessary directories
echo "Creating directories..."
sudo mkdir -p /home/apprunner/sites/logans3d-staging
sudo mkdir -p /etc/logans3d-staging
sudo mkdir -p /var/log/logans3d-staging
sudo chown apprunner:apprunner /home/apprunner/sites/logans3d-staging
sudo chown apprunner:apprunner /var/log/logans3d-staging

# Clone repository (if not already cloned)
if [ ! -d "/home/apprunner/sites/logans3d-staging/.git" ]; then
    echo "Cloning repository..."
    cd /home/apprunner/sites
    git clone https://github.com/yourusername/logans3d-v4.git logans3d-staging
    cd logans3d-staging
else
    echo "Repository already exists, pulling latest..."
    cd /home/apprunner/sites/logans3d-staging
    git pull
fi

# Copy configuration files
echo "Installing configuration files..."
sudo cp deployed-configs/staging/systemd/logans3d-staging.service /etc/systemd/system/
sudo cp deployed-configs/staging/environment /etc/logans3d-staging/
sudo cp deployed-configs/staging/nginx/logans3d-staging.conf /etc/nginx/sites-available/
sudo cp deployed-configs/staging/logrotate/logans3d-staging.conf /etc/logrotate.d/

# Enable nginx site
echo "Enabling nginx configuration..."
sudo ln -sf /etc/nginx/sites-available/logans3d-staging.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# Build application
echo "Building application..."
/usr/local/go/bin/go build -o logans3d ./cmd

# Create data directory for database
echo "Creating data directory..."
mkdir -p /home/apprunner/sites/logans3d-staging/data

# Setup systemd service
echo "Setting up systemd service..."
sudo systemctl daemon-reload
sudo systemctl enable logans3d-staging
sudo systemctl start logans3d-staging

# Setup SSL certificate with Let's Encrypt
echo "Setting up SSL certificate..."
sudo certbot --nginx -d logans3dcreations.digitaldrywood.com --non-interactive --agree-tos --email your-email@example.com

echo "✅ Staging setup complete!"
echo "Check status with: sudo systemctl status logans3d-staging"
echo "View logs with: sudo journalctl -u logans3d-staging -f"