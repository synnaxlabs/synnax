import { PageNavLeaf } from "@/components/PageNav";
import { pythonClientNav } from "@/pages/python-client/nav";
import { rfcNav } from "@/pages/rfc/nav";
import { typescriptClientNav } from "@/pages/typescript-client/nav";

export const pages: PageNavLeaf[] = [
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
  pythonClientNav,
  typescriptClientNav,
  rfcNav,
];
