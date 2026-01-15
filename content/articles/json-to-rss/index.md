---
title: "JSON to RSS feed with N8N and optional HTML manipulation"
date: 2023-08-23
excerpt: "An N8N workflow that converts JSON feeds into RSS format for platforms that don't provide native RSS."
headerAlt: "JSON to RSS conversion"
---

## Overview

This N8N workflow converts JSON feeds into RSS format. A solution for creating RSS feeds from sources that don't provide them natively.

## Key Workflow Steps

The automation performs several functions:

- Fetches JSON data via HTTP request
- Extracts image URLs from description fields using regex
- Strips HTML tags from descriptions
- Generates and serves a new RSS feed
- Enables RSS-based publishing to other platforms

## Implementation Instructions

### Testing Phase

Disable the workflow, deactivate the Feed node, and enable Manual Execution to inspect data flow.

### Production Setup

Disable Manual Execution mode, activate the Feed node, enable the workflow, and access the RSS URL from the Feed node.

I recommend using `view-source:` to inspect RSS content and suggest Feedbro for local testing.

## Technical Components

The solution includes a JavaScript function that:

- Escapes/unescapes HTML entities
- Constructs RSS item elements with metadata
- Generates properly formatted RSS XML output
- Handles datetime conversions to RFC2822 format

The workflow is designed for reusability—you can copy the provided configuration directly into N8N.
