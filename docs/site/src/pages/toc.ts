import { TOCLeaf } from "@/components/TableOfContents";
import { pythonClientTOC } from "@/pages/python-client/toc";
import { rfcTOC } from "@/pages/rfc/toc";
import { typescriptClientTOC } from "@/pages/typescript-client/toc";

export const toc: TOCLeaf[] = [
  {
    name: "Get Started",
    key: "/",
    url: "/",
  },
  {
    name: "Concepts",
    key: "/concepts",
    url: "/concepts",
  },
  {
    key: "/faq",
    url: "/faq",
    name: "FAQ",
  },
  {
    name: "Architecture",
    key: "/architecture",
    url: "/architecture",
  },
  pythonClientTOC,
  typescriptClientTOC,
  rfcTOC,
];
