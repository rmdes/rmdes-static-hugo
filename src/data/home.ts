import { common } from "./common";
import { projects } from "./projects";
import avatar from "../assets/icon.webp";

const home = {
  head: {
    title: "Home",
    description: "General information on me.",
    author: common.author,
    keywords: ["home", "overview"],
    favicon: common.favicon,
    siteTitle: common.title,
    themeColor: common.color,
    lang: common.language
  },
  content: {
    header: {
      name: "Felicitas Pojtinger",
      job: "Cloud Developer / Maker",
      image: avatar,
      follow: "https://twitter.com/intent/user?screen_name=pojntfx",
      email: "mailto:felicitas@pojtinger.com"
    },
    links: [
      {
        title: "CV",
        img:
          "https://img.shields.io/badge/Curriculum%20vitae-PDF-4c85d0.svg?logo=google&logoColor=ffffff",
        link:
          "https://docs.google.com/document/d/1EHisTTIfZj-u1wroThcXz-zHtKwQSPuC79-4W9wPmiI/edit?usp=sharing",
        help: {
          title: "Curriculum Vitae",
          text:
            "A chronological list of all the projects I've worked on (in German).",
          docsLink: "https://en.wikipedia.org/wiki/Curriculum_vitae"
        }
      },
      {
        title: "GitHub",
        img:
          "https://img.shields.io/badge/GitHub-%40pojntfx-181717.svg?logo=github&logoColor=ffffff",
        link: "https://github.com/pojntfx",
        help: {
          title: "GitHub",
          text: "Mirrors of my libre software projects.",
          docsLink: "https://github.com"
        }
      },
      {
        title: "Instagram",
        img:
          "https://img.shields.io/badge/Instagram-%40pojntfx-E4405F.svg?logo=instagram&style=social",
        link: "https://www.instagram.com/pojntfx/",
        help: {
          title: "Instagram",
          text: "Quick and simple, multimedia-driven posts and stories by me.",
          docsLink: "https://www.instagram.com"
        }
      },
      {
        title: "Twitter",
        img:
          "https://img.shields.io/badge/Twitter-%40pojntfx-1DA1F2.svg?logo=twitter&style=social",
        link: "https://twitter.com/pojntfx",
        help: {
          title: "Twitter",
          text: "Opinions and news by me.",
          docsLink: "https://https://twitter.com"
        }
      },
      {
        title: "GitLab",
        img:
          "https://img.shields.io/badge/GitLab-%40pojntfx-fc6d26.svg?logo=gitlab",
        link: "https://gitlab.com/pojntfx",
        help: {
          title: "GitLab",
          text: "Central place for all of my development work.",
          docsLink: "https://gitlab.com"
        }
      },
      {
        title: "Reddit",
        img:
          "https://img.shields.io/badge/reddit-u%2Fpojntfx-FF4500.svg?logo=reddit&style=social",
        link: "https://www.reddit.com/u/pojntfx",
        help: {
          title: "Reddit",
          text: "My account for general discussion.",
          docsLink: "https://www.reddit.com"
        }
      },
      {
        title: "NPM",
        img: "https://img.shields.io/badge/npm-%7epojntfx-CB3837.svg?logo=npm",
        link: "https://www.npmjs.com/~pojntfx",
        help: {
          title: "NPM",
          text: "Most of my development works as NodeJS packages.",
          docsLink: "https://www.npmjs.com"
        }
      },
      {
        title: "Mastodon",
        img:
          "https://img.shields.io/badge/Mastodon-%40pojntfx-3088D4.svg?logo=mastodon&style=social",
        link: "https://mastodon.social/@pojntfx",
        help: {
          title: "Mastodon",
          text: "Discussions and more.",
          docsLink: "https://mastodon.social"
        }
      },
      {
        title: "Facebook",
        img:
          "https://img.shields.io/badge/Facebook-%40pojntfx-3b5998.svg?logo=facebook&style=social",
        link: "https://www.facebook.com/pojntfx",
        help: {
          title: "Facebook",
          text: "Me on Facebook.",
          docsLink: "https://www.facebook.com"
        }
      }
    ],
    skills: {
      description: "I'm a fullstack developer based in Baiersbronn, Germany.",
      descriptionLink:
        "https://www.openstreetmap.org/search?query=Hirschkopfweg%208%20Baiersbronn#map=17/48.50266/8.38205",
      languages: ["Node", "React", "Vue"],
      languagesLink: "https://en.wikipedia.org/wiki/Isomorphic_JavaScript",
      tech: ["FLOSS", "GNU/Linux", "Docker", "Kubernetes"],
      techLink: "https://en.wikipedia.org/wiki/Kubernetes"
    },
    current: "Cluster Platform development for Alpha Horizon.",
    projects,
    activity:
      "https://gateway.activity.demos.clusterplatform.io?username=pojntfx"
  },
  background: common.background
};

export { home };
