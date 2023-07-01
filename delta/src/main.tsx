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
import ReactDOM from "react-dom/client";

import "./index.css";
import { PID } from "./pid/PID/PID";
import { VisLayoutSelectorRenderer } from "./vis/components/VisLayoutSelector";

import { ConnectCluster, useSelectCluster } from "@/cluster";
import { Menu } from "@/components";
import { Docs } from "@/docs";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
} from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { store } from "@/store";
import { useLoadTauriVersion } from "@/version";
import { DefineRange } from "@/workspace";

import "@synnaxlabs/media/dist/style.css";
import "@synnaxlabs/pluto/dist/style.css";

const layoutRenderers = {
  main: LayoutMain,
  connectCluster: ConnectCluster,
  visualization: VisLayoutSelectorRenderer,
  defineRange: DefineRange,
  getStarted: GetStarted,
  docs: Docs,
  pid: PID,
  vis: VisLayoutSelectorRenderer,
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
      workerEnabled
      connParams={cluster?.props}
      workerURL={new URL("./worker.ts", import.meta.url)}
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
