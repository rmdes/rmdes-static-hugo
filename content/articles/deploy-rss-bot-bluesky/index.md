---
title: "Comment déployer un news bot RSS sur Bluesky?"
date: 2023-08-14
excerpt: "A complete guide to deploying an RSS news bot on Bluesky using Docker and bsky.rss."
headerAlt: "RSS bot deployment on Bluesky"
---

## Overview

A guide to deploying an RSS news bot on Bluesky social media platform using Docker and bsky.rss.

## Prerequisites

- Dedicated Bluesky account
- App password credential
- RSS feed source
- bsky.rss repository
- Git installed
- Docker and Docker-Compose installed

## Setup Instructions

### Account & Authentication

Create a dedicated Bluesky account and generate an app password. This tutorial demonstrates deploying a newsbot from Disclose.ngo, an investigative media outlet.

### RSS Feed Selection

Identify your target RSS feed. For the example: English feed at `https://disclose.ngo/feed?lang=en`

### Installation Steps

1. Clone repository: `git clone github.com/milanmdev/bsky.rss disclosengo`
2. Navigate to directory: `cd disclosengo`
3. Copy configuration files from examples
4. Configure docker-compose.yml and data/config.json

## Configuration Details

### Docker Compose Settings

- Memory limits: 256MB max, 128MB reserved
- Environment variables: APP_PASSWORD, INSTANCE_URL, FETCH_URL, IDENTIFIER

### Config Options

- `publishEmbed`: Enable link cards with metadata
- `languages`: Set content language
- `truncate`: Shorten long descriptions
- `runInterval`: Check frequency in seconds

## Deployment

### Testing Phase

Switch to queue branch for queue functionality, build image locally, test with `docker-compose up`

### Production Launch

Return to main branch, pull latest image, deploy with `docker-compose up -d`

## Monitoring Tools

- **Lazydocker**: Container exploration and logging
- **docker-ctop**: Resource monitoring
