#!/usr/bin/env python3
"""
Migration script to convert micro.blog BAR export to Hugo content.
Converts feed.json to articles/ and notes/ with proper frontmatter and aliases.
"""

import json
import os
import re
import shutil
from datetime import datetime
from pathlib import Path
from urllib.parse import urlparse

# Configuration
BAR_DIR = Path("/home/rick/code/bar-blog.rmendes.net")
HUGO_DIR = Path("/home/rick/code/rmdes-static-hugo")
CONTENT_DIR = HUGO_DIR / "content"
STATIC_DIR = HUGO_DIR / "static"

# Old blog domain for alias generation
OLD_DOMAIN = "blog.rmendes.net"


def slugify(text: str) -> str:
    """Convert text to URL-friendly slug."""
    # Remove special chars, lowercase, replace spaces with hyphens
    text = text.lower()
    text = re.sub(r'[^\w\s-]', '', text)
    text = re.sub(r'[-\s]+', '-', text)
    return text.strip('-')[:50]


def extract_slug_from_url(url: str) -> str:
    """Extract slug from micro.blog URL like /2024/01/15/my-post.html"""
    path = urlparse(url).path
    # Remove .html extension and get last part
    slug = path.rstrip('.html').split('/')[-1]
    return slug


def rewrite_image_urls(content: str) -> str:
    """Rewrite absolute blog.rmendes.net/uploads URLs to relative /uploads/"""
    return content.replace(f"https://{OLD_DOMAIN}/uploads/", "/uploads/")


def create_article(item: dict, output_dir: Path) -> None:
    """Create an article (longform post with title)."""
    title = item.get('title', 'Untitled')
    date_str = item.get('date_published', '')
    content_text = item.get('content_text', '')
    content_html = item.get('content_html', '')
    tags = item.get('tags', [])
    url = item.get('url', '')

    # Parse date
    try:
        dt = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        date_formatted = dt.strftime('%Y-%m-%d')
        year = dt.strftime('%Y')
        month = dt.strftime('%m')
        day = dt.strftime('%d')
    except:
        date_formatted = '2024-01-01'
        year, month, day = '2024', '01', '01'

    # Create slug from title or URL
    slug = slugify(title) or extract_slug_from_url(url)
    if not slug:
        slug = f"post-{date_formatted}"

    # Create directory
    article_dir = output_dir / "articles" / slug
    article_dir.mkdir(parents=True, exist_ok=True)

    # Use markdown content, fall back to HTML
    body = rewrite_image_urls(content_text if content_text else content_html)

    # Build aliases for old URL redirects
    old_path = urlparse(url).path
    aliases = [old_path] if old_path else []

    # Filter out 'Longform' and 'Microposts' from tags (they're categories, not tags)
    display_tags = [t for t in tags if t not in ('Longform', 'Microposts')]

    # Build frontmatter
    escaped_title = title.replace('"', '\\"')
    frontmatter = f'''---
title: "{escaped_title}"
date: {date_formatted}
'''
    if display_tags:
        frontmatter += f'tags: {json.dumps(display_tags)}\n'
    if aliases:
        frontmatter += f'aliases: {json.dumps(aliases)}\n'
    frontmatter += '---\n\n'

    # Write file
    with open(article_dir / "index.md", 'w') as f:
        f.write(frontmatter + body)


def create_note(item: dict, output_dir: Path) -> None:
    """Create a note (short post without title)."""
    date_str = item.get('date_published', '')
    content_text = item.get('content_text', '')
    content_html = item.get('content_html', '')
    tags = item.get('tags', [])
    url = item.get('url', '')

    # Parse date
    try:
        dt = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        date_formatted = dt.strftime('%Y-%m-%d')
        time_formatted = dt.strftime('%H%M%S')
    except:
        date_formatted = '2024-01-01'
        time_formatted = '000000'

    # Create slug from date + time to ensure uniqueness
    slug = f"{date_formatted}-{time_formatted}"

    # Create directory
    note_dir = output_dir / "notes" / slug
    note_dir.mkdir(parents=True, exist_ok=True)

    # Use markdown content, fall back to HTML
    body = rewrite_image_urls(content_text if content_text else content_html)

    # Build aliases for old URL redirects
    old_path = urlparse(url).path
    aliases = [old_path] if old_path else []

    # Filter out 'Longform' and 'Microposts' from tags
    display_tags = [t for t in tags if t not in ('Longform', 'Microposts')]

    # Build frontmatter
    frontmatter = f'''---
date: {date_formatted}T{time_formatted[:2]}:{time_formatted[2:4]}:{time_formatted[4:6]}
'''
    if display_tags:
        frontmatter += f'tags: {json.dumps(display_tags)}\n'
    if aliases:
        frontmatter += f'aliases: {json.dumps(aliases)}\n'
    frontmatter += '---\n\n'

    # Write file
    with open(note_dir / "index.md", 'w') as f:
        f.write(frontmatter + body)


def copy_uploads():
    """Copy uploads folder to Hugo static directory."""
    src = BAR_DIR / "uploads"
    dst = STATIC_DIR / "uploads"

    if src.exists():
        if dst.exists():
            print(f"  Uploads directory already exists, merging...")
            # Copy each year folder
            for year_dir in src.iterdir():
                if year_dir.is_dir():
                    dst_year = dst / year_dir.name
                    if dst_year.exists():
                        # Merge files
                        for f in year_dir.iterdir():
                            shutil.copy2(f, dst_year / f.name)
                    else:
                        shutil.copytree(year_dir, dst_year)
        else:
            shutil.copytree(src, dst)
        print(f"  Copied uploads to {dst}")


def main():
    print("=" * 60)
    print("micro.blog BAR to Hugo Migration")
    print("=" * 60)

    # Load feed.json
    feed_path = BAR_DIR / "feed.json"
    with open(feed_path, 'r') as f:
        data = json.load(f)

    items = data['items']
    print(f"\nFound {len(items)} posts in feed.json")

    # Categorize
    articles = [i for i in items if i.get('title')]
    notes = [i for i in items if not i.get('title')]

    print(f"  - {len(articles)} articles (with title)")
    print(f"  - {len(notes)} notes (without title)")

    # Process articles
    print(f"\nMigrating {len(articles)} articles...")
    for i, item in enumerate(articles):
        create_article(item, CONTENT_DIR)
        if (i + 1) % 50 == 0:
            print(f"  Processed {i + 1}/{len(articles)} articles")
    print(f"  Done: {len(articles)} articles created")

    # Process notes
    print(f"\nMigrating {len(notes)} notes...")
    for i, item in enumerate(notes):
        create_note(item, CONTENT_DIR)
        if (i + 1) % 100 == 0:
            print(f"  Processed {i + 1}/{len(notes)} notes")
    print(f"  Done: {len(notes)} notes created")

    # Copy uploads
    print("\nCopying uploads...")
    copy_uploads()

    print("\n" + "=" * 60)
    print("Migration complete!")
    print("=" * 60)
    print("\nNext steps:")
    print("1. Run 'hugo server' to test locally")
    print("2. Check articles at /articles/")
    print("3. Check notes at /notes/")
    print("4. Verify image paths work")
    print("5. Set up cross-domain redirects (see redirect strategy)")


if __name__ == "__main__":
    main()
