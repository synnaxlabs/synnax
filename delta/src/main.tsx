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
import ReactDOM from "react-dom/client";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Docs } from "@/docs";
import { Layout } from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { LinePlot } from "@/lineplot";
import { PID } from "@/pid";
import { store } from "@/store";
import { Version } from "@/version";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";
import { Workspace } from "@/workspace";

import "@/index.css";
import "@synnaxlabs/media/dist/style.css";
import "@synnaxlabs/pluto/dist/style.css";

const layoutRenderers = {
  main: LayoutMain,
  connectCluster: Cluster.Connect,
  visualization: Vis.LayoutSelector,
  defineRange: Workspace.DefineRange,
  getStarted: Layout.GetStarted,
  docs: Docs.Docs,
  vis: Vis.LayoutSelector,
  mosaic: Layout.Mosaic,
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
      [dispatch]
    );
    return [hauled, onHauledChange];
  };

  return (
    <Pluto.Provider
      {...theme}
      workerEnabled
      connParams={cluster?.props}
      workerURL={WorkerURL}
      triggers={triggersProps}
      haul={{ useState: useHaulState }}
    >
      <Vis.Canvas>
        <Layout.Window />
      </Vis.Canvas>
    </Pluto.Provider>
  );
};

const Main = (): ReactElement | null => {
  return (
    <Provider store={store} errorContent={(e) => <h1>{e.message}</h1>}>
      <Layout.RendererProvider value={layoutRenderers}>
        <MainUnderContext />
      </Layout.RendererProvider>
    </Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
