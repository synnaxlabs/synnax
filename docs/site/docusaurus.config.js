// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require("prism-react-renderer/themes/github");
const darkCodeTheme = require("prism-react-renderer/themes/dracula");

const config = {
  title: "Synnax Documentation",
  tagline: "Data Handling for the Modern Era",
  url: "https://docs.synnaxlabs.com/",
  baseUrl: "/",
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",
  favicon: "img//icon/favicon.ico",
  organizationName: "synnaxlabs",
  projectName: "synnax",
  deploymentBranch: "gh-pages",
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },
  presets: [
    [
      "classic",
      ({
        docs: {
          routeBasePath: "/",
          sidebarPath: require.resolve("./sidebars.js"),
          editUrl: "https://github.com/synnaxlabs/synnax/tree/main/docs/site/",
        },
        blog: false,
        theme: {
          customCss: require.resolve("./src/css/custom.css"),
        },
      }),
    ],
  ],

  themeConfig: {
    pages: {
      routeBasePath: "/docs",
    },
    navbar: {
      logo: {
        alt: "Synnax Labs",
        src: "img/logo.svg",
      },
      items: [
        {
          href: "https://github.com/synnaxlabs/synnax",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    footer: {
      style: "light",
      links: [],
      copyright: ".",
    },
    prism: {
      theme: lightCodeTheme,
      darkTheme: darkCodeTheme,
    },
  },
};

module.exports = config;
