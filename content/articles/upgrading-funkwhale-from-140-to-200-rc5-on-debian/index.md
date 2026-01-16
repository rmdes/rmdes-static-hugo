---
title: "Upgrading Funkwhale from 1.4.0 to 2.0.0-rc5 on Debian"
date: 2025-12-29
tags: ["Tech", "Audio"]
aliases: ["/2025/12/29/upgrading-funkwhale-from-to-rc.html"]
---

After running Funkwhale 1.4.0 for a while, I decided to upgrade to the 2.0.0-rc5 release candidate. This post documents the full upgrade process on a Debian 12 (bookworm) server, including the issues I encountered and how to resolve them.
## What's New in Funkwhale 2.0

Funkwhale 2.0 brings significant changes:

- **Libraries replaced by Playlists** - The old "Libraries" concept has been migrated to a more intuitive playlist-based system
- **Python 3.11+ required** - The minimum Python version has been bumped
- **Django 5.1.6** - Major framework upgrade
- **venv instead of virtualenv** - The recommended virtual environment tool has changed
- **Improved artist credits system** - Better handling of artist metadata
- **Federation improvements** - Enhanced ActivityPub support (though this may break compatibility with older instances)

## Prerequisites

Before starting the upgrade, ensure you have:

- Root access to your server
- Sufficient disk space for backups (~2x your database size + config files)
- Python 3.11 installed (`python3 --version`)
- The `python3.11-venv` package installed

```bash
apt install python3.11-venv
```
## Step 1: Create a Full Backup

**Never skip this step.** The database backup is your lifeline if something goes wrong.

### Database Backup

```bash
sudo -u postgres pg_dumpall > /srv/funkwhale/backups/dump_$(date +%d-%m-%Y_%H_%M_%S).sql
```
### Media Backup

If you have a lot of music, consider syncing your media directory to another location:

```bash
# From your local machine (if you have rsync access)
rsync -avz --progress user@yourserver:/srv/funkwhale/data/music ~/funkwhale-backup/music/
```
Or create a compressed archive on the server:

```bash
tar -czvf /srv/funkwhale/backups/media-backup.tar.gz /srv/funkwhale/data/music
```
### Configuration Backup

```bash
cp -a /srv/funkwhale/config /srv/funkwhale/backups/config-backup
```
## Step 2: Stop Funkwhale Services

```bash
systemctl stop funkwhale.target
```
Verify everything is stopped:

```bash
systemctl status funkwhale-server funkwhale-worker funkwhale-beat
```
## Step 3: Backup Current Installation

Before removing anything, backup the current installation for potential rollback:

```bash
BACKUP_DIR="/srv/funkwhale/backups/upgrade-$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

cp -a /srv/funkwhale/api "$BACKUP_DIR/api"
cp -a /srv/funkwhale/front "$BACKUP_DIR/front"
cp -a /srv/funkwhale/virtualenv "$BACKUP_DIR/virtualenv"
cp -a /srv/funkwhale/config "$BACKUP_DIR/config"

# Backup systemd units
mkdir -p "$BACKUP_DIR/systemd"
cp /etc/systemd/system/funkwhale*.service "$BACKUP_DIR/systemd/"
cp /etc/systemd/system/funkwhale.target "$BACKUP_DIR/systemd/"

# Backup nginx config
mkdir -p "$BACKUP_DIR/nginx"
cp /etc/nginx/sites-enabled/your-site.conf "$BACKUP_DIR/nginx/"
cp /etc/nginx/funkwhale_proxy.conf "$BACKUP_DIR/nginx/"
```
## Step 4: Download Funkwhale 2.0.0-rc5

```bash
export FUNKWHALE_VERSION="2.0.0-rc5"
cd /srv/funkwhale

# Download API
curl -L -o "api-$FUNKWHALE_VERSION.zip" \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/-/jobs/artifacts/$FUNKWHALE_VERSION/download?job=build_api"

# Download Frontend
curl -L -o "front-$FUNKWHALE_VERSION.zip" \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/-/jobs/artifacts/$FUNKWHALE_VERSION/download?job=build_front"
```
## Step 5: Install the New Version

### Remove Old Files

```bash
rm -rf /srv/funkwhale/api/*
rm -rf /srv/funkwhale/front/*
```
### Extract New Files

```bash
cd /srv/funkwhale

# Extract API
unzip -q "api-$FUNKWHALE_VERSION.zip" -d extracted
mv extracted/api/* api/
rm -rf extracted "api-$FUNKWHALE_VERSION.zip"

# Extract Frontend
unzip -q "front-$FUNKWHALE_VERSION.zip" -d extracted
mv extracted/front/* front/
rm -rf extracted "front-$FUNKWHALE_VERSION.zip"
```
### Create New Python Virtual Environment

Funkwhale 2.0 uses `venv` instead of `virtualenv`:

```bash
cd /srv/funkwhale

# Remove old virtualenv (if upgrading from 1.x)
rm -rf virtualenv

# Create new venv
python3 -m venv venv
venv/bin/pip install --upgrade pip wheel

# Install Funkwhale API
venv/bin/pip install --editable ./api

# Set correct permissions
chown -R funkwhale:funkwhale api front venv
```
## Step 6: Update Systemd Unit Files

The systemd unit files need to be updated to point to the new `venv` directory instead of `virtualenv`.

You can download them from the repository:

```bash
export FUNKWHALE_VERSION="2.0.0-rc5"

curl -L -o "/etc/systemd/system/funkwhale.target" \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale.target"
curl -L -o "/etc/systemd/system/funkwhale-server.service" \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale-server.service"
curl -L -o "/etc/systemd/system/funkwhale-worker.service" \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale-worker.service"
curl -L -o "/etc/systemd/system/funkwhale-beat.service" \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale-beat.service"

systemctl daemon-reload
```
**Note:** If the curl commands return HTML instead of the service files (404 errors), you may need to clone the repository and copy the files manually:

```bash
git clone --branch 2.0.0-rc5 --depth 1 https://dev.funkwhale.audio/funkwhale/funkwhale.git /tmp/funkwhale-2.0
cp /tmp/funkwhale-2.0/deploy/funkwhale*.service /etc/systemd/system/
cp /tmp/funkwhale-2.0/deploy/funkwhale.target /etc/systemd/system/
systemctl daemon-reload
```
## Step 7: Update Nginx Configuration (Optional)

If you want to use the latest nginx configuration:

```bash
curl -L -o /etc/nginx/funkwhale_proxy.conf \
    "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale_proxy.conf"

nginx -t && systemctl reload nginx
```
## Step 8: Run Database Migrations

This is the critical step that updates your database schema:

```bash
cd /srv/funkwhale

# Collect static files
sudo -u funkwhale venv/bin/funkwhale-manage collectstatic --no-input

# Run migrations
sudo -u funkwhale venv/bin/funkwhale-manage migrate
```
You should see output showing the migrations being applied, including:
- `music.0001_initial` through various migrations
- `playlists` migrations (new in 2.0)
- `common` and `federation` updates

## Step 9: Start Services

```bash
systemctl start funkwhale.target
```
Wait a few seconds and verify everything is running:

```bash
systemctl status funkwhale-server funkwhale-worker funkwhale-beat
```
All three services should show `active (running)`.

## Step 10: Verify the Upgrade

Check that the API is responding:

```bash
curl -s https://your-domain.com/api/v2/ | head -20
```
You should see the API root response with version information.

## Troubleshooting

### Issue: "ensurepip is not available"

If you see this error when creating the venv:

```
Error: Command '['/srv/funkwhale/venv/bin/python3', '-m', 'ensurepip', '--upgrade', '--default-pip']' returned non-zero exit status 1.
```
**Solution:** Install the python3.11-venv package:

```bash
apt install python3.11-venv
rm -rf /srv/funkwhale/venv
python3 -m venv /srv/funkwhale/venv
```
### Issue: Celery Beat Fails to Start

If funkwhale-beat keeps restarting with database errors:

```
_dbm.error: cannot add item to database
```
**Solution:** Remove the corrupted schedule database:

```bash
rm /srv/funkwhale/api/celerybeat-schedule.db
systemctl restart funkwhale-beat
```
### Issue: Systemd Unit Download Returns HTML

The GitLab raw file URLs sometimes don't work for tags. Clone the repo and copy manually (see Step 6).

### Issue: Permission Errors

Always ensure the funkwhale user owns all files:

```bash
chown -R funkwhale:funkwhale /srv/funkwhale/api /srv/funkwhale/front /srv/funkwhale/venv
```
## Rollback Procedure

If something goes seriously wrong, you can rollback:

1. Stop services:
   ```bash
   systemctl stop funkwhale.target
   ```
2. Restore files from backup:
   ```bash
   BACKUP_DIR="/srv/funkwhale/backups/upgrade-YYYYMMDD_HHMMSS"
   rm -rf /srv/funkwhale/api /srv/funkwhale/front /srv/funkwhale/venv
   cp -a "$BACKUP_DIR/api" /srv/funkwhale/
   cp -a "$BACKUP_DIR/front" /srv/funkwhale/
   cp -a "$BACKUP_DIR/virtualenv" /srv/funkwhale/
   ```
3. Restore systemd units:
   ```bash
   cp "$BACKUP_DIR/systemd/"* /etc/systemd/system/
   systemctl daemon-reload
   ```
4. Restore database (if needed):
   ```bash
   sudo -u postgres psql < /srv/funkwhale/backups/dump_DD-MM-YYYY_HH_MM_SS.sql
   ```
5. Start services:
   ```bash
   systemctl start funkwhale.target
   ```
## Automated Upgrade Script

For convenience, I've created a script that automates the entire process with rollback capability. You can find it below:

```bash
#!/bin/bash
#
# Funkwhale Upgrade Script: 1.4.0 -> 2.0.0-rc5
# With rollback capability
#
# Usage: ./upgrade-funkwhale-2.0.sh [upgrade|rollback|status]
#

set -e

# Configuration
FUNKWHALE_VERSION="2.0.0-rc5"
FUNKWHALE_DIR="/srv/funkwhale"
BACKUP_DIR="/srv/funkwhale/backups/upgrade-$(date +%Y%m%d_%H%M%S)"
ROLLBACK_MARKER="/srv/funkwhale/backups/.rollback_info"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root"
        exit 1
    fi
}

check_services_stopped() {
    if systemctl is-active --quiet funkwhale-server.service; then
        log_error "Funkwhale services are still running. Stop them first with: systemctl stop funkwhale.target"
        exit 1
    fi
}

backup_current() {
    log_info "Creating backup in $BACKUP_DIR..."
    mkdir -p "$BACKUP_DIR"

    log_info "Backing up API..."
    cp -a "$FUNKWHALE_DIR/api" "$BACKUP_DIR/api"

    log_info "Backing up frontend..."
    cp -a "$FUNKWHALE_DIR/front" "$BACKUP_DIR/front"

    log_info "Backing up virtualenv..."
    cp -a "$FUNKWHALE_DIR/virtualenv" "$BACKUP_DIR/virtualenv"

    log_info "Backing up config..."
    cp -a "$FUNKWHALE_DIR/config" "$BACKUP_DIR/config"

    log_info "Backing up systemd units..."
    mkdir -p "$BACKUP_DIR/systemd"
    cp /etc/systemd/system/funkwhale*.service "$BACKUP_DIR/systemd/" 2>/dev/null || true
    cp /etc/systemd/system/funkwhale.target "$BACKUP_DIR/systemd/" 2>/dev/null || true

    log_info "Backing up nginx config..."
    mkdir -p "$BACKUP_DIR/nginx"
    cp /etc/nginx/sites-enabled/*.conf "$BACKUP_DIR/nginx/" 2>/dev/null || true
    cp /etc/nginx/funkwhale_proxy.conf "$BACKUP_DIR/nginx/" 2>/dev/null || true

    # Save rollback info
    echo "$BACKUP_DIR" > "$ROLLBACK_MARKER"
    echo "BACKUP_DATE=$(date)" >> "$ROLLBACK_MARKER"
    echo "OLD_VERSION=1.4.0" >> "$ROLLBACK_MARKER"
    echo "NEW_VERSION=$FUNKWHALE_VERSION" >> "$ROLLBACK_MARKER"

    log_info "Backup complete: $BACKUP_DIR"
}

download_new_version() {
    log_info "Downloading Funkwhale $FUNKWHALE_VERSION..."

    cd "$FUNKWHALE_DIR"

    log_info "Downloading API..."
    curl -L -o "api-$FUNKWHALE_VERSION.zip" \
        "https://dev.funkwhale.audio/funkwhale/funkwhale/-/jobs/artifacts/$FUNKWHALE_VERSION/download?job=build_api"

    log_info "Downloading Frontend..."
    curl -L -o "front-$FUNKWHALE_VERSION.zip" \
        "https://dev.funkwhale.audio/funkwhale/funkwhale/-/jobs/artifacts/$FUNKWHALE_VERSION/download?job=build_front"

    log_info "Downloads complete"
}

install_new_version() {
    log_info "Installing Funkwhale $FUNKWHALE_VERSION..."

    cd "$FUNKWHALE_DIR"

    log_info "Removing old API and frontend..."
    rm -rf api/* front/*

    log_info "Extracting API..."
    unzip -q "api-$FUNKWHALE_VERSION.zip" -d extracted
    mv extracted/api/* api/
    rm -rf extracted "api-$FUNKWHALE_VERSION.zip"

    log_info "Extracting frontend..."
    unzip -q "front-$FUNKWHALE_VERSION.zip" -d extracted
    rm -rf front/*
    mv extracted/front/* front/
    rm -rf extracted "front-$FUNKWHALE_VERSION.zip"

    log_info "Creating new Python virtual environment..."
    rm -rf venv
    python3 -m venv venv
    venv/bin/pip install --upgrade pip wheel

    log_info "Installing Funkwhale API package..."
    venv/bin/pip install --editable ./api

    chown -R funkwhale:funkwhale api front venv

    log_info "Installation complete"
}

update_systemd_units() {
    log_info "Updating systemd unit files..."

    curl -L -o "/etc/systemd/system/funkwhale.target" \
        "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale.target"
    curl -L -o "/etc/systemd/system/funkwhale-server.service" \
        "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale-server.service"
    curl -L -o "/etc/systemd/system/funkwhale-worker.service" \
        "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale-worker.service"
    curl -L -o "/etc/systemd/system/funkwhale-beat.service" \
        "https://dev.funkwhale.audio/funkwhale/funkwhale/raw/$FUNKWHALE_VERSION/deploy/funkwhale-beat.service"

    systemctl daemon-reload
    log_info "Systemd units updated"
}

run_migrations() {
    log_info "Running database migrations..."

    cd "$FUNKWHALE_DIR"

    log_info "Collecting static files..."
    sudo -u funkwhale venv/bin/funkwhale-manage collectstatic --no-input

    log_info "Applying database migrations..."
    sudo -u funkwhale venv/bin/funkwhale-manage migrate

    log_info "Migrations complete"
}

start_services() {
    log_info "Starting Funkwhale services..."
    systemctl start funkwhale.target
    systemctl reload nginx

    sleep 5

    if systemctl is-active --quiet funkwhale-server.service; then
        log_info "Funkwhale services started successfully!"
    else
        log_error "Services failed to start. Check logs with: journalctl -u funkwhale-server.service"
        exit 1
    fi
}

do_upgrade() {
    log_info "Starting Funkwhale upgrade to $FUNKWHALE_VERSION"
    log_warn "Make sure you have a database backup before proceeding!"

    check_root
    check_services_stopped

    backup_current
    download_new_version
    install_new_version
    update_systemd_units
    run_migrations
    start_services

    log_info "=========================================="
    log_info "Upgrade complete!"
    log_info "Funkwhale $FUNKWHALE_VERSION is now running"
    log_info "=========================================="
    log_info "If something goes wrong, run: $0 rollback"
}

do_rollback() {
    log_info "Starting rollback..."

    check_root

    if [[ ! -f "$ROLLBACK_MARKER" ]]; then
        log_error "No rollback information found. Cannot rollback."
        exit 1
    fi

    BACKUP_DIR=$(head -1 "$ROLLBACK_MARKER")

    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_error "Backup directory not found: $BACKUP_DIR"
        exit 1
    fi

    log_info "Rolling back from backup: $BACKUP_DIR"

    systemctl stop funkwhale.target 2>/dev/null || true

    log_info "Restoring API..."
    rm -rf "$FUNKWHALE_DIR/api"
    cp -a "$BACKUP_DIR/api" "$FUNKWHALE_DIR/api"

    log_info "Restoring frontend..."
    rm -rf "$FUNKWHALE_DIR/front"
    cp -a "$BACKUP_DIR/front" "$FUNKWHALE_DIR/front"

    log_info "Restoring virtualenv..."
    rm -rf "$FUNKWHALE_DIR/venv" "$FUNKWHALE_DIR/virtualenv"
    cp -a "$BACKUP_DIR/virtualenv" "$FUNKWHALE_DIR/virtualenv"

    log_info "Restoring systemd units..."
    cp "$BACKUP_DIR/systemd/"* /etc/systemd/system/
    systemctl daemon-reload

    log_info "Restoring nginx config..."
    cp "$BACKUP_DIR/nginx/"* /etc/nginx/sites-enabled/ 2>/dev/null || true
    cp "$BACKUP_DIR/nginx/funkwhale_proxy.conf" /etc/nginx/ 2>/dev/null || true

    log_warn "NOTE: Database was NOT rolled back automatically."
    log_warn "To restore database, run:"
    log_warn "  sudo -u postgres psql < /srv/funkwhale/backups/dump_*.sql"

    systemctl start funkwhale.target
    systemctl reload nginx

    log_info "=========================================="
    log_info "Rollback complete!"
    log_info "=========================================="
}

show_status() {
    echo "Current Funkwhale installation:"
    echo "================================"

    if [[ -d "$FUNKWHALE_DIR/venv" ]]; then
        echo "Virtual env: venv (2.0 style)"
        VERSION=$("$FUNKWHALE_DIR/venv/bin/pip" show funkwhale-api 2>/dev/null | grep Version | awk '{print $2}')
    elif [[ -d "$FUNKWHALE_DIR/virtualenv" ]]; then
        echo "Virtual env: virtualenv (1.x style)"
        VERSION=$("$FUNKWHALE_DIR/virtualenv/bin/pip" show funkwhale-api 2>/dev/null | grep Version | awk '{print $2}')
    fi

    echo "Installed version: ${VERSION:-unknown}"
    echo ""
    echo "Service status:"
    systemctl status funkwhale.target --no-pager 2>/dev/null || echo "  Not running"

    if [[ -f "$ROLLBACK_MARKER" ]]; then
        echo ""
        echo "Rollback available:"
        cat "$ROLLBACK_MARKER"
    fi
}

# Main
case "${1:-}" in
    upgrade)
        do_upgrade
        ;;
    rollback)
        do_rollback
        ;;
    status)
        show_status
        ;;
    *)
        echo "Funkwhale Upgrade Script"
        echo "========================"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  upgrade   - Upgrade Funkwhale to $FUNKWHALE_VERSION"
        echo "  rollback  - Rollback to previous version"
        echo "  status    - Show current installation status"
        echo ""
        echo "Before upgrading:"
        echo "  1. Ensure database is backed up"
        echo "  2. Ensure media files are backed up"
        echo "  3. Stop services: systemctl stop funkwhale.target"
        echo ""
        ;;
esac
```
Save this as `upgrade-funkwhale-2.0.sh`, make it executable with `chmod +x upgrade-funkwhale-2.0.sh`, and run with `sudo ./upgrade-funkwhale-2.0.sh upgrade`.

## Conclusion

The upgrade from Funkwhale 1.4.0 to 2.0.0-rc5 went smoothly once I resolved the Python venv package issue and worked around the systemd unit file download problem. The database migrations ran without issues, and all my music library was preserved (now organized under the new playlist system instead of libraries).

If you encounter any issues not covered here, check the [Funkwhale documentation](https://docs.funkwhale.audio/) or join the community on Matrix/IRC for help.

---

*Last updated: December 2025*
