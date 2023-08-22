// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useCallback } from "react";

import { Provider } from "@synnaxlabs/drift/react";
import { Pluto, Haul, Triggers } from "@synnaxlabs/pluto";
import ReactDOM from "react-dom/client";
import { useDispatch } from "react-redux";

import { ConnectCluster, useSelectCluster } from "@/cluster";
import { Docs } from "@/docs";
import {
  LayoutRendererProvider,
  LayoutWindow,
  useThemeProvider,
  GetStarted,
  LayoutMosaic,
  useSelectHauling,
  setHauled,
} from "@/layout";
import { LayoutMain } from "@/layouts/LayoutMain";
import { LinePlot } from "@/line/LinePlot/LinePlot";
import { PID } from "@/pid/PID/PID";
import { store } from "@/store";
import { useLoadTauriVersion } from "@/version";
import { VisCanvas } from "@/vis";
import { VisLayoutSelectorRenderer } from "@/vis/components/VisLayoutSelector";
import WorkerURL from "@/worker?worker&url";
import { DefineRange } from "@/workspace";

import "@/index.css";
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
  line: LinePlot,
  mosaic: LayoutMosaic,
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
  const theme = useThemeProvider();
  useLoadTauriVersion();
  const cluster = useSelectCluster();

  const useHaulState: PureUseState<Hauled[]> = () => {
    const hauled = useSelectHauling();
    const dispatch = useDispatch();
    const onHauledChange = useCallback(
      (hauled: Haul.Item[]) => {
        dispatch(setHauled({ hauling: hauled }));
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
      <VisCanvas>
        <LayoutWindow />
      </VisCanvas>
    </Pluto.Provider>
  );
};

const Main = (): ReactElement | null => {
  return (
    <Provider store={store} errorContent={(e) => <h1>{e.message}</h1>}>
      <LayoutRendererProvider value={layoutRenderers}>
        <MainUnderContext />
      </LayoutRendererProvider>
    </Provider>
  );
};

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Main />);
