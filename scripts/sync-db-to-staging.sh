#!/bin/bash

# Sync local database to staging environment
# Usage: ./scripts/sync-db-to-staging.sh

set -e

# Configuration
LOCAL_DB="./data/database.db"
REMOTE_HOST="jarvis.digitaldrywood.com"
REMOTE_USER="apprunner"
REMOTE_PATH="/home/apprunner/sites/logans3d-staging/data"
REMOTE_DB="${REMOTE_PATH}/database.db"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Syncing local database to staging environment${NC}"

# Check if local database exists
if [ ! -f "$LOCAL_DB" ]; then
    echo -e "${RED}❌ Local database not found at $LOCAL_DB${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Local database size: $(du -h $LOCAL_DB | cut -f1)${NC}"

# Step 1: Backup remote database
echo -e "${YELLOW}📋 Backing up staging database...${NC}"
ssh ${REMOTE_USER}@${REMOTE_HOST} << EOF
    if [ -f "${REMOTE_DB}" ]; then
        echo "Creating backup: ${REMOTE_DB}.backup_${TIMESTAMP}"
        cp "${REMOTE_DB}" "${REMOTE_DB}.backup_${TIMESTAMP}"
        echo "Backup created successfully"

        # Keep only last 5 backups
        ls -t ${REMOTE_DB}.backup_* 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
        echo "Old backups cleaned up (keeping last 5)"
    else
        echo "No existing database found, skipping backup"
    fi
EOF

# Step 2: Stop the staging service to prevent database locks
echo -e "${YELLOW}🛑 Stopping staging service...${NC}"
ssh ${REMOTE_USER}@${REMOTE_HOST} "sudo systemctl stop logans3d-staging"

# Step 3: Upload local database to staging
echo -e "${YELLOW}📤 Uploading local database to staging...${NC}"
scp "$LOCAL_DB" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DB}"

# Step 4: Set correct permissions
echo -e "${YELLOW}🔐 Setting permissions...${NC}"
ssh ${REMOTE_USER}@${REMOTE_HOST} << EOF
    chown apprunner:apprunner "${REMOTE_DB}"
    chmod 644 "${REMOTE_DB}"
EOF

# Step 5: Restart the staging service
echo -e "${YELLOW}🚀 Restarting staging service...${NC}"
ssh ${REMOTE_USER}@${REMOTE_HOST} "sudo systemctl start logans3d-staging"

# Step 6: Verify service is running
echo -e "${YELLOW}✅ Verifying service status...${NC}"
ssh ${REMOTE_USER}@${REMOTE_HOST} "sudo systemctl is-active logans3d-staging"

echo -e "${GREEN}✨ Database sync complete!${NC}"
echo -e "${GREEN}📍 Staging URL: https://logans3dcreations.digitaldrywood.com${NC}"
echo -e "${GREEN}💾 Backup saved as: ${REMOTE_DB}.backup_${TIMESTAMP}${NC}"