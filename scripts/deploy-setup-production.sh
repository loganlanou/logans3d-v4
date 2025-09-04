#!/bin/bash
# Setup script for Logan's 3D production environment

echo "🚀 Setting up Logan's 3D Production Environment..."

# Confirm production deployment
read -p "⚠️  This will setup the PRODUCTION environment. Are you sure? (y/N): " confirm
if [ "$confirm" != "y" ]; then
    echo "Setup cancelled."
    exit 1
fi

# Create necessary directories
echo "Creating directories..."
sudo mkdir -p /home/apprunner/sites/logans3d
sudo mkdir -p /etc/logans3d
sudo mkdir -p /var/log/logans3d
sudo chown apprunner:apprunner /home/apprunner/sites/logans3d
sudo chown apprunner:apprunner /var/log/logans3d

# Clone repository (if not already cloned)
if [ ! -d "/home/apprunner/sites/logans3d/.git" ]; then
    echo "Cloning repository..."
    cd /home/apprunner/sites
    git clone https://github.com/yourusername/logans3d-v4.git logans3d
    cd logans3d
else
    echo "Repository already exists, pulling latest..."
    cd /home/apprunner/sites/logans3d
    git pull
fi

# Copy configuration files
echo "Installing configuration files..."
sudo cp deployed-configs/production/systemd/logans3d.service /etc/systemd/system/
sudo cp deployed-configs/production/environment /etc/logans3d/
sudo cp deployed-configs/production/nginx/logans3d.conf /etc/nginx/sites-available/
sudo cp deployed-configs/production/logrotate/logans3d.conf /etc/logrotate.d/

# Enable nginx site
echo "Enabling nginx configuration..."
sudo ln -sf /etc/nginx/sites-available/logans3d.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# Build application
echo "Building application..."
/usr/local/go/bin/go build -o logans3d ./cmd

# Create data directory for database
echo "Creating data directory..."
mkdir -p /home/apprunner/sites/logans3d/data

# Setup systemd service
echo "Setting up systemd service..."
sudo systemctl daemon-reload
sudo systemctl enable logans3d
sudo systemctl start logans3d

# Setup SSL certificates with Let's Encrypt
echo "Setting up SSL certificates..."
sudo certbot --nginx -d logans3dcreations.com -d www.logans3dcreations.com --non-interactive --agree-tos --email your-email@example.com

echo "✅ Production setup complete!"
echo "Check status with: sudo systemctl status logans3d"
echo "View logs with: sudo journalctl -u logans3d -f"