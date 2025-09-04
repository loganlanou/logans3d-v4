# Production SSL Setup - Final Steps

## Issue: Complete Production SSL Certificate Setup

### Background
The staging environment (logans3dcreations.digitaldrywood.com) is fully deployed with SSL certificates. The production environment is running on port 8007 with temporary HTTP-only nginx configuration, awaiting DNS changes and SSL certificate generation.

### Current Status
- ✅ Production service running on port 8007
- ✅ Temporary HTTP nginx config in place (`/etc/nginx/sites-available/logans3d-temp.conf`)
- ✅ Database migrations completed
- ⏳ Waiting for DNS to point logans3dcreations.com to the server

### Steps to Complete

#### 1. Update DNS Records
Point the following domains to the DigitalOcean server (jarvis.digitaldrywood.com):
- `logans3dcreations.com` - A record pointing to server IP
- `www.logans3dcreations.com` - CNAME pointing to `logans3dcreations.com` or A record to server IP

#### 2. Verify DNS Propagation
Wait for DNS to propagate and verify:
```bash
# Check DNS resolution
dig logans3dcreations.com
dig www.logans3dcreations.com

# Or use nslookup
nslookup logans3dcreations.com
nslookup www.logans3dcreations.com
```

#### 3. Generate SSL Certificate for Production
Once DNS is pointing correctly:
```bash
# SSH into server
ssh -A apprunner@jarvis.digitaldrywood.com

# Generate SSL certificate
sudo certbot --nginx -d logans3dcreations.com -d www.logans3dcreations.com --email YOUR_EMAIL_HERE
```

#### 4. Switch to SSL-enabled Nginx Configuration
After certificate is successfully generated:
```bash
# Remove temporary HTTP-only config
sudo rm /etc/nginx/sites-enabled/logans3d-temp.conf

# Enable the full SSL config
sudo ln -sf /etc/nginx/sites-available/logans3d.conf /etc/nginx/sites-enabled/

# Test nginx configuration
sudo nginx -t

# If test passes, reload nginx
sudo systemctl reload nginx
```

#### 5. Verify Production Site
```bash
# Check HTTPS is working
curl -I https://logans3dcreations.com
curl -I https://www.logans3dcreations.com

# Check service status
sudo systemctl status logans3d

# View logs if needed
sudo journalctl -u logans3d -f
```

### Configuration Files Reference
- **Production SSL nginx config**: `/etc/nginx/sites-available/logans3d.conf`
- **Temporary HTTP config**: `/etc/nginx/sites-available/logans3d-temp.conf`
- **Service file**: `/etc/systemd/system/logans3d.service`
- **Environment vars**: `/etc/logans3d/environment`

### Troubleshooting
If issues arise:
1. Check nginx error logs: `sudo tail -f /var/log/nginx/error.log`
2. Check service logs: `sudo journalctl -u logans3d -n 100`
3. Verify port 8007 is listening: `sudo netstat -tlnp | grep 8007`
4. Test backend directly: `curl http://localhost:8007`

### Future Deployment Commands
After DNS and SSL are set up, use:
```bash
# From local machine
make deploy-production
make log-production
```