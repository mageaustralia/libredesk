#!/bin/bash
set -e

# Libredesk Upgrade Script
# Usage: ./upgrade.sh [version]
# Example: ./upgrade.sh v1.0.1

VERSION=${1:-$(git describe --tags --abbrev=0 origin/main)}
BACKUP_DIR="backups/$(date +%Y%m%d_%H%M%S)"

echo "=== Libredesk Upgrade Script ==="
echo "Target version: $VERSION"
echo ""

# Ensure we're in the right directory
cd /home/ubuntu/libredesk

# Create backup
echo "=== Creating backup ==="
mkdir -p "$BACKUP_DIR"
cp config.toml "$BACKUP_DIR/" 2>/dev/null || true
cp docker-compose.yml "$BACKUP_DIR/"
cp -r uploads "$BACKUP_DIR/" 2>/dev/null || true
pg_dump -h localhost -p 5433 -U libredesk libredesk > "$BACKUP_DIR/database.sql" 2>/dev/null || echo "Warning: Could not backup database"
echo "Backup saved to $BACKUP_DIR"

# Fetch latest from upstream
echo ""
echo "=== Fetching upstream ==="
git fetch origin --tags

# Stash any local changes
echo ""
echo "=== Stashing local changes ==="
git stash push -m "pre-upgrade-$(date +%Y%m%d_%H%M%S)" || true

# Checkout the target version
echo ""
echo "=== Checking out $VERSION ==="
git checkout $VERSION

# Apply custom patches
echo ""
echo "=== Applying custom patches ==="

# Patch 1: Custom TW enhancements (RAG, ReplyBox, etc)
if [ -f custom-patches/0001-feat-Custom-TW-enhancements.patch ]; then
    echo "Applying: 0001-feat-Custom-TW-enhancements.patch"
    git apply --3way custom-patches/0001-feat-Custom-TW-enhancements.patch || {
        echo "WARNING: Patch failed, may need manual resolution"
        echo "Check: git diff and resolve conflicts"
    }
fi

# Patch 2: Ticket ID display fix (sidebar shows name only, header shows full info)
if [ -f custom-patches/0002-ticket-id-display-fix.patch ]; then
    echo "Applying: 0002-ticket-id-display-fix.patch"
    git apply --3way custom-patches/0002-ticket-id-display-fix.patch || {
        echo "WARNING: Patch failed, may need manual resolution"
    }
fi

# Patch 3: Header ticket ID enhancement (add ticket ID to main header)

# Fix docker-compose to use Dockerfile (not Dockerfile.custom)
echo ""
echo "=== Fixing docker-compose.yml ==="
sed -i 's/dockerfile: Dockerfile.custom/dockerfile: Dockerfile/' docker-compose.yml

echo ""
echo "=== Upgrade complete ==="
echo ""
echo "Next steps:"
echo "1. Review changes: git diff"
echo "2. Run deploy: ./deploy.sh"
echo ""
