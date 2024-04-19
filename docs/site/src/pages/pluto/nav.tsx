import { PageNavNode } from "@/components/PageNav";
import { chipStateZ } from "@synnaxlabs/pluto/dist/src/telem/control/aether/chip.js";

export const plutoNav: PageNavNode = {
  key: "pluto",
  name: "Pluto",
  children: [
    {
      name: "Get Started",
      key: "/pluto/plot",
      href: "/pluto/plot",
    },
  ],
};
