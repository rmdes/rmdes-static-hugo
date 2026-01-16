---
title: "How to Bulk Delete Bluesky Posts With Rate Limit Management"
date: 2024-12-15
tags: ["Tech", "Bluesky"]
aliases: ["/2024/12/15/how-to-bulk.html"]
---

In some cases, you may find that you need to bulk-delete many posts from your Bluesky account. For example, perhaps you shared many links to a particular domain and now you want to remove them en masse. Doing this manually would be tedious. Fortunately, we can automate the process using a script written in TypeScript.


This script leverages the official `@atproto/api` package to:
1. Log into your Bluesky account.
2. Fetch all posts that match certain criteria (e.g., containing a specific domain in their facets, embeds, or entities).
3. Delete them in batches while respecting and reacting to rate limits.


## Key Features


- **Domain-based Filtering:**  
  The script identifies posts containing a specific domain by checking:
  - Facets with `app.bsky.richtext.facet#link`.
  - External embeds with `app.bsky.embed.external`.
  - Legacy entities with `type: link`.
  
- **Rate Limit Management (Proactive):**  
  The Bluesky PDS imposes a rate limit of 5000 deletion points per hour. Deletions cost 1 point each. The script proactively monitors how many deletions it has performed within the current hour. When it approaches the limit, it waits until the hour has elapsed before continuing.


- **Rate Limit Management (Fallback):**  
  If the script ever hits a `429 Rate Limit Exceeded` error, it will parse the `ratelimit-reset` header and wait until the given time before retrying that batch of deletions. This ensures that if the proactive limit check is not enough, the script still handles the server’s instructions gracefully.


- **Batch Operations and Delays:**  
  To avoid rapid-fire requests, the script:
  - Performs deletions in configurable batch sizes (default: 200 per batch).
  - Waits a short delay between batches to spread requests out over time.


## Prerequisites


- **Node.js and npm:**  
  Ensure you have a recent version of Node.js installed.


- **Install Dependencies:**
  ```bash
  npm install @atproto/api p-ratelimit
  ```
 
## Credentials


Replace your-handle and your-password in the script with your Bluesky account credentials. You should only do this with an account you control and trust running scripts on.


## Running the Script


Save the script below as bluesky-sweep.ts.


Run it using:


```bash
npx ts-node bluesky-sweep.ts
```
## Configuration Parameters
- TARGET_DOMAIN: Set this to the domain you want to search for in your posts.
- DELETES_PER_BATCH: Number of posts per deletion batch.
- MAX_DELETES_PER_HOUR: Maximum deletions allowed per hour (5000 is the current default from Bluesky).
- SAFE_MARGIN: A buffer to start waiting before hitting the exact limit.
- DELAY_BETWEEN_BATCHES_MS: Milliseconds to wait between each batch.


```typescript
/**
 * Bulk Delete Bluesky Posts with Domain Filtering and Rate Limit Management
 *
 * This script:
 * - Logs in to a Bluesky account.
 * - Fetches posts containing a specified domain via facets, embeds, or entities.
 * - Deletes them in batches, respecting and reacting to rate limits.
 *
 * Adjust the constants below to fit your needs before running.
 */


import { BskyAgent } from '@atproto/api';
import { pRateLimit } from 'p-ratelimit';


const VERBOSE = false;
const TARGET_DOMAIN = 'futura-sciences.com';


// Known limit and configurations
const MAX_DELETES_PER_HOUR = 5000;
const DELETES_PER_BATCH = 200;
const DELAY_BETWEEN_BATCHES_MS = 5000; // 5 seconds between batches
const SAFE_MARGIN = 100; // Start waiting before we hit exactly 5000


(async () => {
  const agent = new BskyAgent({ service: 'https://bsky.social' });
  await agent.login({
    identifier: 'your-handle',
    password: 'your-password',
  });


  console.log(`Logged in as ${agent.session!.handle} (${agent.session!.did})`);


  const limit = pRateLimit({ concurrency: 3, interval: 1000, rate: 5 });


  const getRecordId = (uri: string) => {
    const idx = uri.lastIndexOf('/');
    return uri.slice(idx + 1);
  };


  const chunked = <T>(arr: T[], size: number): T[][] => {
    const chunks: T[][] = [];
    for (let idx = 0; idx < arr.length; idx += size) {
      chunks.push(arr.slice(idx, idx + size));
    }
    return chunks;
  };


  const sleep = (ms: number) => new Promise((res) => setTimeout(res, ms));


  let deletes: any[] = [];
  let cursor: string | undefined;
  let batchCount = 0;


  // Fetch posts
  do {
    console.log(`Fetching records (cursor: ${cursor || 'none'})...`);
    const response = await limit(() =>
      agent.api.com.atproto.repo.listRecords({
        repo: agent.session!.did,
        collection: 'app.bsky.feed.post',
        limit: 100,
        cursor,
        reverse: true,
      })
    );


    cursor = response.data.cursor;
    batchCount++;
    console.log(`Processing batch #${batchCount}, ${response.data.records.length} records fetched`);


    for (const record of response.data.records) {
      if (VERBOSE) console.log(`\nChecking record URI: ${record.uri}`);
      const val = record.value as any;


      let found = false;


      // Check facets for links
      const facets = val?.facets || [];
      for (const facet of facets) {
        const features = facet.features || [];
        for (const feature of features) {
          if (feature.$type === 'app.bsky.richtext.facet#link' && feature.uri.includes(TARGET_DOMAIN)) {
            if (VERBOSE) console.log(`Found target domain in facet link: ${feature.uri}`);
            found = true;
            break;
          }
        }
        if (found) break;
      }


      // Check embed if not found yet
      if (!found && val?.embed) {
        const embed = val.embed;
        if (embed.$type === 'app.bsky.embed.external' && embed.external?.uri?.includes(TARGET_DOMAIN)) {
          if (VERBOSE) console.log(`Found target domain in embed: ${embed.external.uri}`);
          found = true;
        }
      }


      // Check entities (legacy) if not found yet
      if (!found && val?.entities && Array.isArray(val.entities)) {
        for (const entity of val.entities) {
          if (entity.type === 'link' && entity.value.includes(TARGET_DOMAIN)) {
            if (VERBOSE) console.log(`Found target domain in entities link: ${entity.value}`);
            found = true;
            break;
          }
        }
      }


      if (found) {
        deletes.push({
          $type: 'com.atproto.repo.applyWrites#delete',
          collection: 'app.bsky.feed.post',
          rkey: getRecordId(record.uri),
        });
      }
    }
  } while (cursor);


  console.log(`\nFound ${deletes.length} posts containing '${TARGET_DOMAIN}'`);


  if (deletes.length === 0) {
    console.log('No posts to delete.');
    return;
  }


  const chunkedDeletes = chunked(deletes, DELETES_PER_BATCH);
  console.log(`Deletion can be done in ${chunkedDeletes.length} batched operations`);


  let hourWindowStart = Date.now();
  let deletesThisHour = 0;


  for (let idx = 0; idx < chunkedDeletes.length; idx++) {
    const chunk = chunkedDeletes[idx];


    // Check if we're near the hourly limit
    if (deletesThisHour + chunk.length > (MAX_DELETES_PER_HOUR - SAFE_MARGIN)) {
      const now = Date.now();
      const elapsed = now - hourWindowStart;
      const oneHourMs = 3600000;


      if (elapsed < oneHourMs) {
        const waitTime = oneHourMs - elapsed;
        console.log(`Approaching hourly limit. Waiting ${Math.ceil(waitTime / 60000)} minutes to reset.`);
        await sleep(waitTime);
      }


      hourWindowStart = Date.now();
      deletesThisHour = 0;
    }


    console.log(`Deleting batch #${idx + 1} with ${chunk.length} posts...`);


    // Retry loop in case of rate limit errors
    let success = false;
    while (!success) {
      try {
        await limit(() =>
          agent.api.com.atproto.repo.applyWrites({
            repo: agent.session!.did,
            writes: chunk,
          })
        );
        console.log(`Batch operation #${idx + 1} completed`);
        success = true;
      } catch (error: any) {
        if (error.status === 429) {
          console.warn('Rate limit exceeded, checking headers to wait until reset...');
          const resetTimeStr = error.headers?.['ratelimit-reset'];
          let waitSeconds = 60; // default wait


          if (resetTimeStr) {
            const resetTime = parseInt(resetTimeStr, 10);
            const now = Math.floor(Date.now() / 1000);
            const diff = resetTime - now;
            if (diff > 0) {
              waitSeconds = diff;
            }
          }


          console.log(`Waiting ${waitSeconds} seconds before retrying...`);
          await sleep(waitSeconds * 1000);
          console.log('Retrying this batch...');
        } else {
          console.error(`Error performing batch #${idx + 1}:`, error);
          // If it's a non-rate-limit error, stop the process
          break;
        }
      }
    }


    if (success) {
      deletesThisHour += chunk.length;
      await sleep(DELAY_BETWEEN_BATCHES_MS);
    } else {
      // If we failed without rate limit handling, break out
      break;
    }
  }


  console.log('Done');
})();


```
## Notes

### Rate Limits 
- https://docs.bsky.app/docs/advanced-guides/rate-limits
### Post Structure
- https://github.com/bluesky-social/atproto/blob/main/lexicons/app/bsky/feed/post.json
### Gist
- https://gist.github.com/rmdes/4e3e7b85d8bd58b419240def8dd21252
### Code
- https://github.com/rmdes/bluesky-deleter

