// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StrictMode, useEffect, useState } from "react";

import { Theming } from "@synnaxlabs/pluto";
import "@synnaxlabs/pluto/dist/style.css";
import ReactDOM from "react-dom/client";

import { MainLayout } from "@/components";

import { Provider, useDispatch } from "react-redux";

import { ConnectCluster } from "@/features/cluster";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
  maybeCreateGetStartedTab,
} from "@/features/layout";
import { useLoadTauriVersion } from "@/features/version";
import { VisLayoutRenderer } from "@/features/vis";
import { DefineRange } from "@/features/workspace";

import { store as promise } from "./store";

import "./index.css";

const layoutRenderers = {
  main: MainLayout,
  connectCluster: ConnectCluster,
  visualization: VisLayoutRenderer,
  defineRange: DefineRange,
  getStarted: GetStarted,
};

const MainUnderContext = (): JSX.Element => {
  const d = useDispatch();
  const theme = useThemeProvider();
  useLoadTauriVersion();
  useEffect(() => {
    d(maybeCreateGetStartedTab());
  }, []);
  return (
    <Theming.Provider {...theme}>
      <LayoutRendererProvider value={layoutRenderers}>
        <LayoutWindow />
      </LayoutRendererProvider>
    </Theming.Provider>
  );
};

const Main = (): JSX.Element | null => {
  const [store, setStore] = useState<any | null>(null);
  useEffect(() => {
    promise.then((s) => setStore(s)).catch(console.error);
  }, []);
  if (store == null) return null;
  return (
    <StrictMode>
      <Provider store={store}>
        <MainUnderContext />
      </Provider>
    </StrictMode>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
