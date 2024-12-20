// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@synnaxlabs/media/dist/media.css";
import "@synnaxlabs/pluto/dist/pluto.css";

import { Provider } from "@synnaxlabs/drift/react";
import {
  type Color,
  type Haul,
  Pluto,
  type state,
  type Triggers,
} from "@synnaxlabs/pluto";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type ReactElement, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Confirm } from "@/confirm";
import { Docs } from "@/docs";
import { ErrorOverlayWithoutStore, ErrorOverlayWithStore } from "@/error/Overlay";
import { LabJack } from "@/hardware/labjack";
import { NI } from "@/hardware/ni";
import { OPC } from "@/hardware/opc";
import { Task } from "@/hardware/task";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { Layouts } from "@/layouts";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Ontology } from "@/ontology";
import { Permissions } from "@/permissions";
import { Range } from "@/range";
import { Schematic } from "@/schematic";
import { SERVICES } from "@/services";
import { store } from "@/store";
import { Table } from "@/table";
import { User } from "@/user";
import { Version } from "@/version";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";
import { Workspace } from "@/workspace";

const LAYOUT_RENDERERS: Record<string, Layout.Renderer> = {
  ...Layouts.LAYOUTS,
  ...Docs.LAYOUTS,
  ...Workspace.LAYOUTS,
  ...Schematic.LAYOUTS,
  ...LinePlot.LAYOUTS,
  ...LabJack.LAYOUTS,
  ...OPC.LAYOUTS,
  ...Range.LAYOUTS,
  ...Cluster.LAYOUTS,
  ...NI.LAYOUTS,
  ...Channel.LAYOUTS,
  ...Version.LAYOUTS,
  ...Confirm.LAYOUTS,
  ...Label.LAYOUTS,
  ...User.LAYOUTS,
  ...Permissions.LAYOUTS,
  ...Log.LAYOUTS,
  ...Task.LAYOUTS,
  ...Table.LAYOUTS,
};

const CONTEXT_MENU_RENDERERS: Record<string, Layout.ContextMenuRenderer> = {
  ...Schematic.CONTEXT_MENUS,
  ...LinePlot.CONTEXT_MENUS,
};

const PREVENT_DEFAULT_TRIGGERS: Triggers.Trigger[] = [
  ["Control", "P"],
  ["Control", "Shift", "P"],
  ["Control", "MouseLeft"],
  ["Control", "W"],
];

const TRIGGERS_PROVIDER_PROPS: Triggers.ProviderProps = {
  preventDefaultOn: PREVENT_DEFAULT_TRIGGERS,
  preventDefaultOptions: { double: true },
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

const useColorContextState: state.PureUse<Color.ContextState> = () => {
  const colorContext = Layout.useSelectColorContext();
  const dispatch = useDispatch();
  const onColorContextChange = useCallback(
    (state: Color.ContextState) => dispatch(Layout.setColorContext({ state })),
    [dispatch],
  );
  return [colorContext, onColorContextChange];
};

const useBlockDefaultDropBehavior = (): void =>
  useEffect(() => {
    const doc = document.documentElement;
    doc.addEventListener("dragover", (e) => e.preventDefault());
    doc.addEventListener("drop", (e) => e.preventDefault());
    return () => {
      doc.removeEventListener("dragover", (e) => e.preventDefault());
      doc.removeEventListener("drop", (e) => e.preventDefault());
    };
  }, []);

const MainUnderContext = (): ReactElement => {
  const theme = Layout.useThemeProvider();
  const cluster = Cluster.useSelect();
  const activeRange = Range.useSelect();
  useBlockDefaultDropBehavior();
  return (
    <QueryClientProvider client={client}>
      <Pluto.Provider
        theming={theme}
        channelAlias={{
          // Set the alias active range to undefined if the range is not saved in Synnax,
          // otherwise it will try to pull aliases from a range that doesn't exist.
          activeRange: activeRange?.persisted ? activeRange.key : undefined,
        }}
        workerEnabled
        connParams={cluster ?? undefined}
        workerURL={WorkerURL}
        triggers={TRIGGERS_PROVIDER_PROPS}
        haul={{ useState: useHaulState }}
        color={{ useState: useColorContextState }}
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

export const Console = (): ReactElement => (
  <ErrorOverlayWithoutStore>
    <Provider store={store}>
      <ErrorOverlayWithStore>
        <Layout.RendererProvider value={LAYOUT_RENDERERS}>
          <Layout.ContextMenuProvider value={CONTEXT_MENU_RENDERERS}>
            <Ontology.ServicesProvider services={SERVICES}>
              <MainUnderContext />
            </Ontology.ServicesProvider>
          </Layout.ContextMenuProvider>
        </Layout.RendererProvider>
      </ErrorOverlayWithStore>
    </Provider>
  </ErrorOverlayWithoutStore>
);
