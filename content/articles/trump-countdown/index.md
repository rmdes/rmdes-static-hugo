---
title: "Un compteur pour déterminer quand Trump va être dégagé (ou pas)"
date: 2025-01-26
excerpt: "A JavaScript automation script for N8N that calculates and publishes daily countdowns for major U.S. political events."
headerAlt: "Trump countdown automation"
---

## Overview

A JavaScript automation script designed for N8N that calculates and publishes daily countdowns for major U.S. political events.

## Target Events

- **Midterms:** November 3, 2026
- **Presidential Election:** November 7, 2028
- **Inauguration Day:** January 20, 2029

## Functionality

The script computes remaining days until each event and generates visual progress bars using block characters (█ for filled, ▒ for empty).

### Sample Output

"There are 646 days until the Midterms, 1381 days until the next Presidential Election, and 1490 days until the next Inauguration Day" with corresponding progress bars showing completion percentages.

## Implementation

- N8N JavaScript function node
- Scheduled daily execution at 14:00 CET (8:00 ET)
- Output formatted as JSON
- Published via Bluesky to [@trumpwatch.skyfleet.blue](https://bsky.app/profile/trumpwatch.skyfleet.blue)

## Technical Details

Calculates elapsed days since Trump's January 20, 2025 inauguration to determine progress percentages across three election-related milestones.
