// Scratch page for visual options. Not for merge.

import "@synnaxlabs/lyra/dist/base.css";
import "@/index.css";
import "@/options/options.css";

import { Haul } from "@synnaxlabs/lyra/haul";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement } from "react";
import ReactDOM from "react-dom/client";

import { BaselineSection } from "@/options/sections/Baseline";
import { CheckboxSection } from "@/options/sections/Checkboxes";
import { KeycapsSection } from "@/options/sections/Keycaps";
import { ModalSection } from "@/options/sections/Modal";
import { OverflowSection } from "@/options/sections/Overflow";
import { PaletteSection } from "@/options/sections/Palette";
import { RadiusSection } from "@/options/sections/Radius";
import { TableInputSection } from "@/options/sections/TableInput";
import { TabStyleSection } from "@/options/sections/TabStyle";
import { TextSizeSection } from "@/options/sections/TextSize";
import { WeightSection } from "@/options/sections/Weight";

const PAGE_THEME = { key: "synnaxLight" };

const Page = (): ReactElement => (
  <main className="opt-page">
    <header className="opt-page-head">
      <h1>Synnax component options</h1>
      <p>
        Real Lyra and Pluto components at real sizes, in the dark and light themes side
        by side. The baseline is the only render with font smoothing off; every option
        below it, A included, has smoothing on. Letters are stable: refer to an option
        as 4C, 7B, and so on.
      </p>
    </header>
    <BaselineSection />
    <RadiusSection />
    <OverflowSection />
    <TextSizeSection />
    <KeycapsSection />
    <WeightSection />
    <ModalSection />
    <TabStyleSection />
    <TableInputSection />
    <PaletteSection />
    <CheckboxSection />
  </main>
);

const Main = (): ReactElement => (
  <Triggers.Provider>
    <Tooltip.Config>
      <Haul.Provider>
        <Theming.Provider theme={PAGE_THEME}>
          <Page />
        </Theming.Provider>
      </Haul.Provider>
    </Tooltip.Config>
  </Triggers.Provider>
);

ReactDOM.createRoot(document.getElementById("root")!).render(<Main />);
