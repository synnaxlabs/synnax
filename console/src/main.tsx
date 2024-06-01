// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@synnaxlabs/media/dist/style.css";
import "@synnaxlabs/pluto/dist/style.css";

import { Provider } from "@synnaxlabs/drift/react";
import { type Haul, Pluto, type state,type Triggers } from "@synnaxlabs/pluto";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactElement, useCallback } from "react";
import ReactDOM from "react-dom/client";
import { useDispatch } from "react-redux";

import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { ErrorOverlay } from "@/error/Overlay";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Layout } from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { LinePlot } from "@/lineplot";
import { Ontology } from "@/ontology";
import { Range } from "@/range";
import { Schematic } from "@/schematic";
import { SERVICES } from "@/services";
import { store } from "@/store";
import { Version } from "@/version";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";
import { Workspace } from "@/workspace";

const layoutRenderers: Record<string, Layout.Renderer> = {
  main: LayoutMain,
  ...Docs.LAYOUTS,
  ...Layout.LAYOUTS,
  ...Vis.LAYOUTS,
  ...Workspace.LAYOUTS,
  ...Schematic.LAYOUTS,
  ...LinePlot.LAYOUTS,
  ...OPC.LAYOUTS,
  ...Range.LAYOUTS,
  ...Cluster.LAYOUTS,
  ...NI.LAYOUTS,
  ...Channel.LAYOUTS,
  ...Version.LAYOUTS,
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

const useHaulState: state.PureUse<Haul.DraggingState> = () => {
  const hauled = Layout.useSelectHauling();
  const dispatch = useDispatch();
  const onHauledChange = useCallback(
    (state: Haul.DraggingState) => dispatch(Layout.setHauled(state)),
    [dispatch],
  );
  return [hauled, onHauledChange];
};

const MainUnderContext = (): ReactElement => {
  const theme = Layout.useThemeProvider();
  const cluster = Cluster.useSelect();
  const activeRange = Range.useSelect();
  return (
    <QueryClientProvider client={client}>
      <Pluto.Provider
        theming={theme}
        channelAlias={{
          // Set the alias active range to undefined if the range is not saved in Synnax,
          // otherwise it will try to pull aliases from a range that doesn't exist.
          activeRange: activeRange?.persisted ? activeRange?.key : undefined,
        }}
        workerEnabled
        connParams={cluster?.props}
        workerURL={WorkerURL}
        triggers={triggersProps}
        haul={{ useState: useHaulState }}
        alamos={{
          level: "debug",
          include: [],
        }}
      >
        <Vis.Canvas>
          <Layout.Window />
        </Vis.Canvas>
      </Pluto.Provider>
    </QueryClientProvider>
  );
};

const Main = (): ReactElement => (
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

const rootEl = document.getElementById("root") as unknown as HTMLElement;

ReactDOM.createRoot(rootEl).render(<Main />);
