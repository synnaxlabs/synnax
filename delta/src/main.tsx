// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, StrictMode } from "react";

import { Provider } from "@synnaxlabs/drift";
import { Menu as PMenu, Pluto } from "@synnaxlabs/pluto";
import { appWindow } from "@tauri-apps/api/window";
import ReactDOM from "react-dom/client";

import "./index.css";

import { ConnectCluster, useSelectCluster } from "@/cluster";
import { Menu } from "@/components";
import { DocsLayoutRenderer } from "@/docs";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
} from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { store } from "@/store";
import { useLoadTauriVersion } from "@/version";
import { VisLayoutRenderer } from "@/vis";
import { DefineRange } from "@/workspace";

import "@synnaxlabs/media/dist/style.css";
import "@synnaxlabs/pluto/dist/style.css";

const layoutRenderers = {
  main: LayoutMain,
  connectCluster: ConnectCluster,
  visualization: VisLayoutRenderer,
  defineRange: DefineRange,
  getStarted: GetStarted,
  docs: DocsLayoutRenderer,
};

export const DefaultContextMenu = (): ReactElement => (
  <PMenu>
    <Menu.Item.HardReload />
  </PMenu>
);

const MainUnderContext = (): ReactElement => {
  const theme = useThemeProvider();
  const menuProps = PMenu.useContextMenu();
  useLoadTauriVersion();
  const cluster = useSelectCluster();
  return (
    <Pluto
      {...theme}
      workerEnabled={appWindow.label === "main"}
      params={cluster?.props}
    >
      <PMenu.ContextMenu menu={() => <DefaultContextMenu />} {...menuProps}>
        <LayoutWindow />
      </PMenu.ContextMenu>
    </Pluto>
  );
};

const Main = (): ReactElement | null => {
  return (
    <StrictMode>
      <Provider store={store} errorContent={(e) => <h1>{e.message}</h1>}>
        <LayoutRendererProvider value={layoutRenderers}>
          <MainUnderContext />
        </LayoutRendererProvider>
      </Provider>
    </StrictMode>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
