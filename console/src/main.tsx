// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useCallback } from "react";

import { Provider } from "@synnaxlabs/drift/react";
import { Pluto, type Haul, type Triggers, type state } from "@synnaxlabs/pluto";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ReactDOM from "react-dom/client";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { ErrorOverlay } from "@/error/Overlay";
import { Layout } from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { Mosaic } from "@/layouts/mosaic";
import { LinePlot } from "@/lineplot";
import { Ontology } from "@/ontology";
import { PID } from "@/pid";
import { Range } from "@/range";
import { SERVICES } from "@/services";
import { store } from "@/store";
import { Version } from "@/version";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";
import { Workspace } from "@/workspace";

import "@/index.css";
import "@synnaxlabs/media/dist/style.css";
import "@synnaxlabs/pluto/dist/style.css";

const layoutRenderers: Record<string, Layout.Renderer> = {
  main: LayoutMain,
  connectCluster: Cluster.Connect,
  visualization: Vis.LayoutSelector,
  defineRange: Range.EditLayout,
  getStarted: Layout.GetStarted,
  docs: Docs.Docs,
  vis: Vis.LayoutSelector,
  mosaic: Mosaic.Window,
  createWorkspace: Workspace.Create,
  [LinePlot.LAYOUT_TYPE]: LinePlot.LinePlot,
  [PID.LAYOUT_TYPE]: PID.PID,
};

const PREVENT_DEFAULT_TRIGGERS: Triggers.Trigger[] = [
  ["Control", "P"],
  ["Control", "Shift", "P"],
  ["Control", "MouseLeft"],
];

const triggersProps: Triggers.ProviderProps = {
  preventDefaultOn: PREVENT_DEFAULT_TRIGGERS,
};

const client = new QueryClient();

const MainUnderContext = (): ReactElement => {
  const theme = Layout.useThemeProvider();
  Version.useLoadTauri();
  const cluster = Cluster.useSelect();

  const useHaulState: state.PureUse<Haul.DraggingState> = () => {
    const hauled = Layout.useSelectHauling();
    const dispatch = useDispatch();
    const onHauledChange = useCallback(
      (state: Haul.DraggingState) => {
        dispatch(Layout.setHauled(state));
      },
      [dispatch],
    );
    return [hauled, onHauledChange];
  };

  const activeRange = Range.useSelect();

  return (
    <QueryClientProvider client={client}>
      <Pluto.Provider
        {...theme}
        channelAlias={{ activeRange: activeRange?.key }}
        workerEnabled
        connParams={cluster?.props}
        workerURL={WorkerURL}
        triggers={triggersProps}
        haul={{ useState: useHaulState }}
        alamos={{
          level: "info",
          include: ["aether.telem"],
        }}
      >
        <Vis.Canvas>
          <Layout.Window />
        </Vis.Canvas>
      </Pluto.Provider>
    </QueryClientProvider>
  );
};

const Main = (): ReactElement | null => {
  return (
    <Provider store={store}>
      <ErrorOverlay>
        <Layout.RendererProvider value={layoutRenderers}>
          <Ontology.ServicesProvider services={SERVICES}>
            <MainUnderContext />
          </Ontology.ServicesProvider>
        </Layout.RendererProvider>
      </ErrorOverlay>
    </Provider>
  );
};

const rootEl = document.getElementById("root") as unknown as HTMLElement;

ReactDOM.createRoot(rootEl).render(<Main />);
