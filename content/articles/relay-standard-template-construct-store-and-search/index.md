---
title: "Relay : Standard Template Construct: Store and Search The Entirety of Human Knowledge"
date: 2023-04-23
tags: ["Tech", "Random"]
aliases: ["/2023/04/23/relay-standard-template.html"]
---

I'm storing this here because people need to find this and have access to this and help relay this, it's too important that knowledge be shared and accessible. 

Pasted from Reddit Scihub sub
https://www.reddit.com/r/scihub/comments/12detqs/standard_template_construct_store_and_search_the/

We are creating [free, unblockable, and easily clonable library](http://standard-template-construct.org/) for both people and machines, living entirely in IPFS and working without any centralised server. It's already functional and provides access to many contemporary scholarly works that are missing from other free libraries.

The site does not host any illegal content - there is no any site in a classical meaning - but just guides you through what already has been stored in IPFS :)

**History**

Even leaving aside these already well-known claims that knowledge should be free, I was personally insulted when saw something like this:

[Would you pay 40€ for the names of potato overlords? Are they afraid to be death noted and hiding behind the paywall? ](https://preview.redd.it/s0g4mcyjj8sa1.jpg?width=2300&format=pjpg&auto=webp&v=enabled&s=7454e1995eedb7fed154768c16a4e82e22e93d9b)

But formally, we have always been keeping in mind that

* Unrestricted access to all knowledge is necessary for emerging new (semi-)digital lifeforms, such as AI and cyborgs
* The free flow of information promotes growth, while restriction leads to stagnation and starvation
* Withholding knowledge further perpetuates inequal opportunities amongst individuals and nations
* In the event of potential armageddon, private knowledge will not survive. However, freely replicating knowledge will serve as resurrection points for fallen civilizations
* Colonizing new worlds will require a reliable replication of knowledge, which is hardened by copyright

All these reasons have led to the creation of Nexus, the search engine and the group of people with the same name aimed at fighting against copyright in science through the most famous tool available when other ones are failing - the riot, namely digital riot. We also wanted to experiment with search and distributed technologies, and, of course, we had a desire to read.

Three years ago, we created a Telegram bot called Nexus that mirrors books and scientific publications from Library Genesis. Telegram of 2020 was a welcoming house, and we spent the past three years productively. The bot acquired the capability to provide access to new papers in the end of 2021, which was especially useful as Sci-Hub stopped doing so, linked Z-Library books in 2022, and even fought against a stupid Indian publishing house that bombarded us with claims in 2022/23 (the traces of the fight are remembered by the name of the currently working Telegram bots). Over time, Nexus Bots became a convenient way to find papers, given the blocks happening here and there on the wild Web.

[TorrentFreak about us, unwillingly confused with Z-Library](https://preview.redd.it/ycwwsn4sj8sa1.png?width=1638&format=png&auto=webp&v=enabled&s=e892bc8e0e062b20d90f767b7b0e1de988562e76)

But the most important changes are about to become a reality. Here, I would like to summarise all that we have done and highlight essential details that our users are still unaware of. We have never had enough time to promote ourselves rapidly (the team is small, sometimes small to the most degenerate size possible), and I hope this post will answer several asked questions about Nexus.

**Inside the Nexus Search**

Bots are just the face of Nexus and every your search query is served by the search engine, the heart with the name Nexus Search. Originally, the first versions of Nexus Search were based on Postgres tables, but their search capabilities were too sad. As a result, we used a dedicated search software that has been the core of Nexus for years, namely [Summa](https://github.com/izihawa/summa).

This is the reason why Nexus does not publish SQL dumps like LibGen - there are no such dumps because there is no SQL server behind - but publishes data dumps in binary format of Summa that can be [downloaded](https://stdtc-eth.ipns.dweb.link/#/replication), [attached](https://izihawa.github.io/summa/blog/mitigating-internet-censorship-and-privacy-issues/) and used directly in this search engine.

If you want to explore the landscape of scholarly publications, you can freely use Nexus Search as a replacement for the [CrossRef Database](https://www.crossref.org/blog/2022-public-data-file-of-more-than-134-million-metadata-records-now-available/). It does not contain all fields that exist in CrossRef but provides the most important ones required for search, such as title, authors, abstract, journal name, ISSN/ISBN/DOI identifiers. Nexus Search also contains cross-references, allowing users to explore graph properties of scholarly publications. And not a single asshole such as Elsevier will be [capable to interrupt you](https://www.techdirt.com/articles/20151117/09383132839/elsevier-says-downloading-content-mining-licensed-copies-research-papers-could-be-considered-stealing.shtml).

Summa search engine provides many opportunities for researchers. It allows you to perform aggregation queries to conduct analytics, and export documents into JSON much faster than any SQL database if you want to use data in other databases. In addition, we have calculated PageRank for all papers up to 2023 year and extracted text layers for top-downloaded 350k articles, providing valuable data for researchers.

Our next goal is to reliably distribute scholar data across the globe using IPFS, after which we plan to extract all text layers. Together with search, it will open doors for people to train AI on various scientific topics or even on the entire corpus of science.

Additionally, Nexus Search contains IPFS content identifiers (CIDs) for almost all books and for a lot of recent papers that are absent in many known libraries. CIDs make possible for users to download PDFs directly through the IPFS network. The collection of CIDs nowadays are not very large but constantly growing up.  


>*A several words about IPFS, a system similar to BitTorrent. With IPFS, you can get an identifier for your file and pass it to other people, who can then download the file directly from your computer using the identifier. After that, they can become a source of the file for others too. This means that files can spread like a* *~~digital~~* *disease. This is exactly what we need to ensure that knowledge is replicated reliably. If we can put scholarly papers on IPFS, store their identifiers to the search database and then put the search database on IPFS too, and then ensure that people use and replicate Nexus Search on their computers, our goal of knowledge preservation will be achieved.*

**Important:** Nexus Search does not contain CIDs for *all* scimag items, though we are going to add them. Our focus is on items, absent in scimag and other collections: recent papers and the most important papers of the past.

Nexus Search is a valuable database, but users need a way to access its information. While it is possible to [download and deploy](http://standard-template-construct.org/#/replication) the data on a local search server, this process is not always comfortable for most people. Therefore, Nexus Search requires a front-end. The original method of accessing Nexus Search was through already mentioned Nexus Bots.

**A Breed Of Nexus Bots**

Bots exist on Telegram, and you must be a registered user of this messenger to use them. Legal restrictions have compelled us to remove direct links to the bots, but they can be easily found if you navigate through our social media pages (linked at the end).

Bots are widely used, can be cloned, and provide a rather rich query syntax to request what you need. The only remaining issue is that it is still Telegram, a proprietary and closed system that may ban us or disappear one day.

To mitigate our dependency on the proprietary system, we have started to build Standard Template Construct on the basis of existing distributed technologies to ensure that the flow of knowledge will never stop.

**Standard Template Construct**

Named after the imaginary computational machine of the Warhammer 40,000 universe, Standard Template Construct (STC) is an IPFS-backed web-site that touted as the most reliable way to access knowledge. Here is a brief introduction to what it is, how to use it, and why it is important.

Nexus Search, the main database behind STC, was already put to IPFS for replication purposes. But we have moved further and used a search engine that also may be executed within static sites, doing all search operations inside your browser. Such a search engine can be put to IPFS too and access Nexus Search living in IPFS.  


>In the field of IT, there is a concept of a network drive, which acts like a physical drive but is accessed over a network. Most databases store their files on drives and can store files on network drives as well. IPFS can also be considered as a large, reliable and slow network drive, which means, in theory, that any database can be launched over IPFS. In practice, this depends on how the database accesses its files. For instance, if the database reads many small intervals of its files, this pattern can be inefficient because every read is translated to a network request. We have worked hard to reduce the number of reads in our database, making STC usable over network drives and IPFS in practice.

This means that all the necessary components to access Nexus Search are available in IPFS. There is no need for a dedicated search server or website to execute your search request, and therefore accessing papers can be done **without the involvement of centralized services**. All you have to do is to open STC through IPFS, perform a search, and then to download your book or paper through its CID that is also stored in Nexus Search.

[Who entirely replicated STC since several days after release](https://preview.redd.it/n0ah8dewj8sa1.jpg?width=1629&format=pjpg&auto=webp&v=enabled&s=33accdd682c9b637a49896b4a0c393c06fc05a18)

By using STC you are also becoming a seeder of database and stored items. It happens because of the way how IPFS is working: after getting a data, your IPFS instance starts to distribute it further.

This also means that no one will be able to collect logs on your search queries or sell your personal data to a third party. IPFS makes you the owner of the database and search engine, significantly enhancing your privacy.

**How to Access STC**

Before you begin, please keep in mind that we are still in the process of developing STC. We've done our best to reduce network requirements, but you should be aware that opening the 70GB search database will require you to download 20-30MB on your first visit after which it will be cached, and 0.5-1.5MB for each search thereafter.

It's also worth noting that we do not currently support Safari versions prior to 16.4, and we have not tested STC on every possible browser. Unfortunately, it is also required to disable Brave Shields (if you use Brave browser) as it breaks subdomain queries that are normal for many sites. Refresh page if it hangs (and report us about it).

The first thing you need to know is that STC is a static site, meaning it doesn't require any API or remote server. Once you've retrieved the site files through IPFS, your browser can render the site, and you can start making search queries. All search requests are served locally or through interaction with IPFS.

So the main issue is how to receive STC to your browser. In general, IPFS supposes two antagonistic ways of how you may open static sites hosted on it:

*Public Gateway*

IPFS is a separate network and to access it, you need to have [IPFS Desktop installed](https://docs.ipfs.tech/install/ipfs-desktop/). To address the case when you have no way to install IPFS locally (e.g. iPhone, restricted environments, casual users who do not care about this fancy technology), the authors invested in creating public gateways that allow to access IPFS through HTTP. You may use such a public gateway without installing IPFS, but you should remember that there are many other users who think the same way, creating incredible pressure on public gateway sites. Hence, gateways usually work very slowly. Moreover, [public gateways are the first subjects for censorship](https://torrentfreak.com/cloudflare-disables-access-to-pirated-content-on-its-ipfs-gateway-230324/).

*Local IPFS Daemon*

The more preferable option is to [install IPFS Desktop](https://docs.ipfs.tech/install/ipfs-desktop/) and [IPFS Companion](https://docs.ipfs.tech/install/ipfs-companion/). Then, you will have direct access to IPFS, which is more performant and reliable.

Now, after you have chosen between Public Gateway and Local IPFS Daemon, you are ready to open STC, but before that, you need to learn its name in IPFS network. IPFS provides non-human-readable identifiers called CIDs for all stored entities which are unique for every stored entity and also is generated automatically.

Every particular version of the STC has its own identifier. It's important to underline that CIDs are immutable, so visiting the site by its CID means you will get the same version of the site every time. But what if we add new features or upload new papers to STC? New versions of STC will have new CIDs and to help you find the actual CIDs of new versions, we have registered the domain name: [standard-template-construct.org](http://standard-template-construct.org)

Here you are, opened the Standard Template Construct and ready to read. Put in the search bar DOI or text query, navigate through what you have found and download it.

[Recent versions of STC](https://preview.redd.it/c6ixygoog8sa1.png?width=2412&format=png&auto=webp&v=enabled&s=5c6fac36824f55a4e03fe35ebc1f893656e89b14)

**How to help us?**

*Easy mode*

* You may join [our Telegram](https://t.me/nexus_search) where we publish DOI-CIDs lists which is enough to help us with seeding and to start your own library. People from South America, Asia and Africa are especially welcomed because now STC is under-replicated on these continents.  
Pin can be done as following:  
`cat doi-cids.txt | xargs -L1 bash -c 'ipfs pin add $1'`
* Pin STC itself (papers not included, \~70GB) and **repin it periodically**:  
`ipfs pin add /ipns/standard-template-construct.org`
* Donate, we have a tip jar at Donate page in STC: [http://standard-template-construct.org/#/donate](http://standard-template-construct.org/#/donate)

*Hard mode*

* If you have an access to scholarly publications (no matter how - via scimag torrents, direct access, or any other means), you can organise seeding yourself and provide us with a list of DOI-CIDs in text format (`<lowercased-non-escaped-doi> <cid>\n`) that we will merge into STC.  
The condition is that you will stay and seed your collection for a while, and only DOI-stamped items will be accepted. Files should be chunked and hashed using the following parameters:  
`ipfs add --hash=blake3 --chunker=size-1048576`  
As mentioned earlier, Nexus Search does not contain all CIDs of the scimag collection. However, if you are a SciHub torrent seeder, you can unpack scimag zip files, hash them individually, and organize their seeding. Such DOI-CID lists will be accepted to STC, but keep in mind that a single IPFS instance will hardly be capable of managing more than 1-2 million individual files.
* [Deploy](http://standard-template-construct.org/#/replication) your own Nexus Search server, do search queries and pin sub-collections of your interest

**Battle for the Free Knowledge is Already Going On**

The Internet Archive, Z-Library, and Sci-Hub are currently facing legal challenges, leaving us to question the world we want to live in. Do we want a world dominated by superficial entertainment and corporate/government censorship, or do we aspire to a scientific utopia with a free flow of knowledge?

https://preview.redd.it/8uslt93w8asa1.jpg?width=1000&format=pjpg&auto=webp&v=enabled&s=e8a4478549068e2fd4e5eaa5b07e52b3c5591576

The answer is clear. We need to protect free libraries as they are a breath of fresh air in our contemporary web. Let's work together to spread the word and support this vital cause. Remember, the Web was created to share knowledge, not to restrict it.

**Contacts**

[Our Twitter](https://twitter.com/the_superpirate)

[Our Mastodon](https://kolektiva.social/@the_superpirate)

[Our Telegram](https://t.me/nexus_search)

Email: [**ultranymous@proton.me**](mailto:ultranymous@proton.me)
