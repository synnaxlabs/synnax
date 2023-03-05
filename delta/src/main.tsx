// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StrictMode } from "react";

import { Provider } from "@synnaxlabs/drift";
import "@synnaxlabs/media/dist/style.css";
import { Theming, Triggers, Menu as PMenu } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import ReactDOM from "react-dom/client";

import { LayoutMain } from "./layouts/LayoutMain";
import { store as promise } from "./store";

import { Menu } from "@/components";
import { ConnectCluster } from "@/features/cluster";
import { DocsLayoutRenderer } from "@/features/docs";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
} from "@/features/layout";
import { useLoadTauriVersion } from "@/features/version";
import { VisLayoutRenderer } from "@/features/vis";
import { DefineRange } from "@/features/workspace";

import "./index.css";

const layoutRenderers = {
  main: LayoutMain,
  connectCluster: ConnectCluster,
  visualization: VisLayoutRenderer,
  defineRange: DefineRange,
  getStarted: GetStarted,
  docs: DocsLayoutRenderer,
};

export const DefaultContextMenu = (): JSX.Element => (
  <PMenu>
    <Menu.Item.HardReload />
  </PMenu>
);

const MainUnderContext = (): JSX.Element => {
  const theme = useThemeProvider();
  useLoadTauriVersion();
  const menuProps = PMenu.useContextMenu();
  return (
    <Theming.Provider {...theme}>
      <Triggers.Provider>
        <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
          <LayoutRendererProvider value={layoutRenderers}>
            <LayoutWindow />
          </LayoutRendererProvider>
        </PMenu.ContextMenu>
      </Triggers.Provider>
    </Theming.Provider>
  );
};

const Main = (): JSX.Element | null => (
  <StrictMode>
    <Provider store={promise}>
      <MainUnderContext />
    </Provider>
  </StrictMode>
);

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
