---
title: "Get ready to quit #Reddit but keep an eye on specific Users or Subs while the Exode is taking place with RSS"
date: 2023-06-19
tags: ["Tech", "Photos"]
aliases: ["/2023/06/19/get-ready-to.html"]
---



Let's say you want to keep an eye on Reddit, without using the Reddit app and without having to login to Reddit, the solution is RSS feeds.

## RSS feed for a Subreddit

The RSS feed for a subreddit can be accessed using the following URL format:

`https://www.reddit.com/r/[subreddit name]/.rss`

Just replace [subreddit name] with the name of the subreddit you want to monitor. For example, if you wanted to get the RSS feed for the subreddit "news", you would use the following URL:

`https://www.reddit.com/r/news/.rss`

This URL will give you an XML document that you can feed into an RSS reader to receive updates about new posts in the subreddit.

## Posts of a particular user

The RSS feed for a Reddit user's posts can be accessed using the following URL format:

`https://www.reddit.com/user/[username]/.rss`

Simply replace `[username]` with the actual Reddit username. For example, for the Reddit CEO, whose username is "spez​", the RSS feed for his posts would be:

`https://www.reddit.com/user/spez​/.rss`

Please note that Reddit must have RSS feeds enabled for this to work. Also, the user's posts must be public. If the user's posts are private or if Reddit has disabled RSS feeds, this URL will not return an RSS feed.

## Track the comments of the same user

The RSS feed for a Reddit user's **comments** can be accessed using the following URL format:

`https://www.reddit.com/user/spez/comments/.rss`

Just replace `[username]` with the actual Reddit username. So, Huffman, whose username is "spez​", the RSS feed for his comments would be:

`https://www.reddit.com/user/spez​/comments/.rss`

Again, this will only work if Reddit has RSS feeds enabled and if the user's comments are public.

## RSS Feed Readers

Yes, that's still a thing ! I have been using it since...2003 !!

Check out this [page](https://alternativeto.net/category/books--news/rss-feed-reader/
) to find one for iOs, Android, Linux or Windows

My preference are divided in two kinds : 

### Self-Hosted 
- FreshRSS (my choice)
- Tiny Tiny RSS (advanced features)
- Miniflux (minimal)

### Web based services
- Inoreader (my choice, advanced features)
- Feedly

Since I'm on Android I use **FeedMe** to connect to my self-hosted **FreshRSS** instance and If I need to use **Inoreader**, I use the Inoreader android app. 

## How to find the RSS feed of any particular website ? 

This website has all you need to know : https://openrss.org/blog

My own approach is to use [Feedbro](https://nodetics.com/feedbro/) with Feedbro you can install the extension on your browser and if there is an RSS feed wherever you are on the web, you'll be able to find it without having to memorize how RSS works and how to get it, Feedbro will find it for you !

<img src="uploads/2023/rmdes-digital-art-wide-angle-shot-reddit-user-disg" width="600" height="300" alt="">
