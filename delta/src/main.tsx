// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StrictMode } from "react";

import { Provider as DriftProvider } from "@synnaxlabs/drift";
import { Theming } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import ReactDOM from "react-dom/client";

import { MainLayout } from "@/components";
import { ConnectCluster } from "@/features/cluster";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
} from "@/features/layout";
import { useLoadTauriVersion } from "@/features/version";
import { VisualizationLayoutRenderer } from "@/features/visualization";
import { DefineRange } from "@/features/workspace";
import { store } from "@/store";

import "./index.css";

const layoutRenderers = {
  main: MainLayout,
  connectCluster: ConnectCluster,
  visualization: VisualizationLayoutRenderer,
  defineRange: DefineRange,
};

const MainUnderContext = (): JSX.Element => {
  const theme = useThemeProvider();
  useLoadTauriVersion();
  return (
    <Theming.Provider {...theme}>
      <LayoutRendererProvider value={layoutRenderers}>
        <LayoutWindow />
      </LayoutRendererProvider>
    </Theming.Provider>
  );
};

const Main = (): JSX.Element => (
  <StrictMode>
    <DriftProvider store={store}>
      <MainUnderContext />
    </DriftProvider>
  </StrictMode>
);

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
