---
title: "JSON to RSS feed with N8N and optional HTML manipulation"
date: 2023-08-23
tags: ["Coding"]
aliases: ["/2023/08/23/json-to-rss.html"]
---

I'm so thrilled to finally having done it right !

This workflow that you can copy and use on N8N does a few things
- HTTP request a JSON feed to get last published items at the source (belgian AFSCA)
- Using regex to pick image file from the description field
- Clean up of the Description field from any html tags
- Serve a new RSS feed (Afsca does not have one) 
- Use the RSS feed to Publish anywhere

### Testing your feed
- Disable the workflow
- Disable the Feed node (left webhook)
- Enable Manual Execution
- Run the Workflow to inspect it and pin data if you need to test it
 
### To enable the servicing of the RSS feed
- Disable the Manual Execution node
- Enable the Feed node (left webhook)
- Enable the workflow
- Visit your production RSS URL provided by the Feed node
  
Use view-source:https://URL of your RSS feed to easily inspect the content of your rss feed.
You can use Feedbro to test your RSS feed locally

<img src="uploads/2023/2023-08-23-23-08.png" width="600" height="198" alt="">

## Copy into N8N

- Select the entire content of this block code below and paste it inside N8N
- you will get the node and everything like in the image above. 
- the Function node showcased below does not need copying, it's already included in the workflow. 

```json
{
  "meta": {
    "instanceId": "58fd5c3ff393ef21f618201de491e9a03a72661d4848a2ad337fc80dc260a4d9"
  },
  "nodes": [
    {
      "parameters": {},
      "id": "c7da8a26-baa3-47e5-a3c5-9f886d67dce4",
      "name": "When clicking \"Execute Workflow\"",
      "type": "n8n-nodes-base.manualTrigger",
      "typeVersion": 1,
      "position": [
        460,
        760
      ],
      "disabled": true
    },
    {
      "parameters": {
        "fieldToSplitOut": "items",
        "options": {}
      },
      "id": "042e6e0d-11ff-46e9-b49e-f440f44d6fa2",
      "name": "Split out lists",
      "type": "n8n-nodes-base.itemLists",
      "position": [
        1120,
        600
      ],
      "typeVersion": 1
    },
    {
      "parameters": {
        "path": "afsca.rss",
        "responseMode": "responseNode",
        "options": {}
      },
      "id": "bb82d17d-e134-40be-8f85-898600577ff5",
      "name": "Feed",
      "type": "n8n-nodes-base.webhook",
      "position": [
        460,
        460
      ],
      "webhookId": "63f12265-8387-4bb7-bef0-d7eb93e49e11",
      "typeVersion": 1
    },
    {
      "parameters": {
        "respondWith": "text",
        "responseBody": "={{ $json[\"data\"] }}",
        "options": {
          "responseCode": 200,
          "responseHeaders": {
            "entries": [
              {
                "name": "Content-Type",
                "value": "application/rss+xml"
              }
            ]
          }
        }
      },
      "id": "e6857661-3b79-4c8b-a73d-c67ef4aebd9d",
      "name": "Serve feed",
      "type": "n8n-nodes-base.respondToWebhook",
      "position": [
        2040,
        600
      ],
      "typeVersion": 1
    },
    {
      "parameters": {
        "url": "https://www.inoreader.com/stream/user/1005072895/tag/Afsca/view/json",
        "sendHeaders": true,
        "headerParameters": {
          "parameters": [
            {
              "name": "accept",
              "value": "application/json"
            }
          ]
        },
        "options": {}
      },
      "id": "4361bdc9-2795-4ac5-99eb-efc4db72a64c",
      "name": "HTTP Request",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 4.1,
      "position": [
        820,
        600
      ]
    },
    {
      "parameters": {
        "functionCode": "const escapeHTML = str => {\n    if (!str) return \"\";\n    return str.replace(/[&<>'\"]/g, \n        tag => ({\n            '&': '&',\n            '<': '<',\n            '>': '>',\n            \"'\": ''',\n            '\"': '"'\n        }[tag])\n    );\n};\n\nconst unescapeHTML = str => {\n    if (!str) return \"\";\n    return str.replace(/(<|>|"|'|&)/g, \n        tag => ({\n            '<': '<',\n            '>': '>',\n            '"': '\"',\n            ''': \"'\",\n            '&': '&'\n        }[tag])\n    );\n};\n\nlet feedItems = [];\nfor (item of items) {\n    feedItems.push(`<item>\n        <title><![CDATA[${unescapeHTML(item.json.Title)}]]></title>\n        <guid isPermaLink=\"false\">${item.json.guid}</guid>\n        <media:content url=\"${item.json.Image}\" type=\"image/jpeg\" />\n        <link>${item.json.Link}</link>\n        <pubDate>${DateTime.fromISO(item.json.Date).toRFC2822()}</pubDate>\n        <description><![CDATA[${unescapeHTML(item.json.Description || \"\")}]]></description>\n    </item>`);\n}\n\nconst feedTitle = \"RappelConso\";  // Set this to your desired feed title\nconst feedDescription = \"Rappel Conso monitoring.\";  // Set this to your desired feed description\nconst feedLink = \"https://rmendes.net\";  // Set this to your main RSS page or main website URL\n\nreturn [{\n    data: `<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rss xmlns:dc=\"http://purl.org/dc/elements/1.1/\" xmlns:media=\"http://search.yahoo.com/mrss/\" version=\"2.0\">\n    <channel>\n        <title><![CDATA[${feedTitle}]]></title>\n        <link>${feedLink}</link>\n        <description><![CDATA[${feedDescription}]]></description>\n        <pubDate>${DateTime.fromISO(item.json.Date).toRFC2822()}</pubDate>\n        ${feedItems.join('\\n')}\n    </channel>\n</rss>`\n}];\n\n"
      },
      "id": "986ea46f-4206-431e-92a9-199072492648",
      "name": "Define feed items1",
      "type": "n8n-nodes-base.function",
      "position": [
        1840,
        600
      ],
      "typeVersion": 1
    },
    {
      "parameters": {
        "values": {
          "string": [
            {
              "name": "ImageURL",
              "value": "={{$json[\"content_html\"].match(/src=\"([^\"]+)\"/) ? $json[\"content_html\"].match(/src=\"([^\"]+)\"/)[1] : \"\"}}\n"
            },
            {
              "name": "CleanDesc",
              "value": "={{$json[\"content_html\"].replace(/<\\/?[^>]+(>|$)/g, \" \").trim().replace(/\\s\\s+/g, ' ').substring(0,200)}}\n"
            }
          ]
        },
        "options": {}
      },
      "id": "b1f7e9b0-3270-4a44-bfdf-5fa7519e9d6b",
      "name": "Fetch IMG + CleanDesc",
      "type": "n8n-nodes-base.set",
      "typeVersion": 2,
      "position": [
        1360,
        600
      ]
    },
    {
      "parameters": {
        "keepOnlySet": true,
        "values": {
          "string": [
            {
              "name": "Title",
              "value": "={{ $json.title }}"
            },
            {
              "name": "Link",
              "value": "={{ $json.url }}"
            },
            {
              "name": "Date",
              "value": "={{ $json.date_published }}"
            },
            {
              "name": "guid",
              "value": "={{ $json.id }}"
            },
            {
              "name": "Description",
              "value": "={{ $json.CleanDesc }}"
            },
            {
              "name": "Image",
              "value": "={{ $json.ImageURL }}"
            }
          ]
        },
        "options": {}
      },
      "id": "5cdbd79f-6f5b-42ee-8f3d-b8d1de320949",
      "name": "Set Everything",
      "type": "n8n-nodes-base.set",
      "typeVersion": 2,
      "position": [
        1620,
        600
      ]
    }
  ],
  "connections": {
    "When clicking \"Execute Workflow\"": {
      "main": [
        [
          {
            "node": "HTTP Request",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Split out lists": {
      "main": [
        [
          {
            "node": "Fetch IMG + CleanDesc",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Feed": {
      "main": [
        [
          {
            "node": "HTTP Request",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "HTTP Request": {
      "main": [
        [
          {
            "node": "Split out lists",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Define feed items1": {
      "main": [
        [
          {
            "node": "Serve feed",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Fetch IMG + CleanDesc": {
      "main": [
        [
          {
            "node": "Set Everything",
            "type": "main",
            "index": 0
          }
        ]
      ]
    },
    "Set Everything": {
      "main": [
        [
          {
            "node": "Define feed items1",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
```
## Javascript function node used in this workflow
- I have escapeHTML and unEscapeHTML for each use case, this is completely optional
 and pretty much there for me to have both use cases at hand when I work with feeds. 

{{< highlight js "linenos=table,hl_lines=8 225-250,linenostart=252" >}}
const escapeHTML = str => {
    if (!str) return "";
    return str.replace(/[&<>'"]/g, 
        tag => ({
            '&': '&amp;',
            '<': '&lt;',
            '>': '&gt;',
            "'": '&#39;',
            '"': '&quot;'
        }[tag])
    );
};

const unescapeHTML = str => {
    if (!str) return "";
    return str.replace(/(&lt;|&gt;|&quot;|&#39;|&amp;)/g, 
        tag => ({
            '&lt;': '<',
            '&gt;': '>',
            '&quot;': '"',
            '&#39;': "'",
            '&amp;': '&'
        }[tag])
    );
};

let feedItems = [];
for (item of items) {
    feedItems.push(`<item>
        <title><![CDATA[${unescapeHTML(item.json.Title)}]]></title>
        <guid isPermaLink="false">${item.json.guid}</guid>
        <media:content url="${item.json.Image}" type="image/jpeg" />
        <link>${item.json.Link}</link>
        <pubDate>${DateTime.fromISO(item.json.Date).toRFC2822()}</pubDate>
        <description><![CDATA[${unescapeHTML(item.json.Description || "")}]]></description>
    </item>`);
}

const feedTitle = "RappelConso";  // Set this to your desired feed title
const feedDescription = "Rappel Conso monitoring.";  // Set this to your desired feed description
const feedLink = "https://rmendes.net";  // Set this to your main RSS page or main website URL

return [{
    data: `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:media="http://search.yahoo.com/mrss/" version="2.0">
    <channel>
        <title><![CDATA[${feedTitle}]]></title>
        <link>${feedLink}</link>
        <description><![CDATA[${feedDescription}]]></description>
        <pubDate>${DateTime.fromISO(item.json.Date).toRFC2822()}</pubDate>
        ${feedItems.join('\n')}
    </channel>
</rss>`
}];
{{< / highlight >}}


