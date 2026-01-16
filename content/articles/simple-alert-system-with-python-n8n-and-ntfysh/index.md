---
title: "Simple alert system with python, N8N and Ntfy.sh "
date: 2023-07-10
tags: ["Tech", "Photos"]
aliases: ["/2023/07/10/simple-alert-system.html"]
---

So today, with the assistance of #openAI ChatGPT I built a python script to monitor my servers for errors, tasks that do not finish well, containers that become unresponsive etc..

It use python to monitor logs in the filesystem of the Host, when some error match the defined events it triggers a #n8n webhook that relay the JSON payload to ntfy.sh

I have then ntfy.sh mobile app with me or on the desktop and I'm subscribed to Topics that allow me to categorize the event errors and from a glance on my phone, have a pretty neat idea of what's happening in 2 seconds, allowing me to react quickly and adapt to the situation.

It was a fun learning experience to prompt ChatGPT to do exactly what I wanted, by giving enough constraints, examples of my log files and doing some use case research to create matching scenarios and iterate in a few hours something really cool and fun to use.!

I'll tweak the code a bit more and then I'll publish it on github. 



<img src="uploads/2023/cb6bfa50c5.jpg" width="600" height="450" alt="Landscape form wallonia just for fun! ">
