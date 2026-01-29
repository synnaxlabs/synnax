// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/index.css";
import "@synnaxlabs/media/dist/media.css";

// import "@synnaxlabs/pluto/dist/pluto.css";
import { Provider } from "@synnaxlabs/drift/react";
import {
  type Color,
  type Haul,
  Pluto,
  preventDefault,
  type state,
  Synnax,
  type Triggers,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useMemo } from "react";
import { useDispatch } from "react-redux";

import { Access } from "@/access";
import { Arc } from "@/arc";
import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Code } from "@/code";
import { Arc as ArcCode } from "@/code/arc";
import { Lua } from "@/code/lua";
import { COMMANDS } from "@/commands";
import { CSV } from "@/csv";
import { Docs } from "@/docs";
import { Errors } from "@/errors";
import { EXTRACTORS } from "@/extractors";
import { Hardware } from "@/hardware";
import { FILE_INGESTORS } from "@/ingestors";
import { Label } from "@/label";
import { Layout } from "@/layout";
import { Layouts } from "@/layouts";
import { LinePlot } from "@/lineplot";
import { Log } from "@/log";
import { Modals } from "@/modals";
import { Ontology } from "@/ontology";
import { Palette } from "@/palette";
import { Range } from "@/range";
import { Schematic } from "@/schematic";
import { SERVICES } from "@/services";
import { Status } from "@/status";
import { store } from "@/store";
import { Table } from "@/table";
import { User } from "@/user";
import { Version } from "@/version";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";
import { Workspace } from "@/workspace";

const LAYOUT_RENDERERS: Record<string, Layout.Renderer> = {
  ...Channel.LAYOUTS,
  ...Cluster.LAYOUTS,
  ...CSV.LAYOUTS,
  ...Docs.LAYOUTS,
  ...Hardware.LAYOUTS,
  ...Label.LAYOUTS,
  ...Layouts.LAYOUTS,
  ...LinePlot.LAYOUTS,
  ...Log.LAYOUTS,
  ...Modals.LAYOUTS,
  ...Range.LAYOUTS,
  ...Schematic.LAYOUTS,
  ...Table.LAYOUTS,
  ...User.LAYOUTS,
  ...Version.LAYOUTS,
  ...Vis.LAYOUTS,
  ...Workspace.LAYOUTS,
  ...Arc.LAYOUTS,
  ...Status.LAYOUTS,
  ...Access.LAYOUTS,
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
    doc.addEventListener("dragover", preventDefault);
    doc.addEventListener("drop", preventDefault);
    return () => {
      doc.removeEventListener("dragover", preventDefault);
      doc.removeEventListener("drop", preventDefault);
    };
  }, []);

const ArcLSPClientSetter = ({ children }: { children: ReactElement }): ReactElement => {
  const client = Synnax.use();
  const monaco = Code.useMonaco();
  useEffect(() => {
    // Only start LSP when Monaco is initialized and client is available
    if (monaco == null) return;
    void ArcCode.setSynnaxClient(client);
  }, [client, monaco]);
  return children;
};

const MainUnderContext = (): ReactElement => {
  const theme = Layout.useThemeProvider();
  const cluster = Cluster.useSelect();
  useBlockDefaultDropBehavior();

  const monacoExtensions = useMemo(() => [...Lua.EXTENSIONS], []);
  const monacoServices = useMemo(() => [...Lua.SERVICES, ...ArcCode.SERVICES], []);

  return (
    <Pluto.Provider
      theming={theme}
      workerEnabled
      connParams={cluster ?? undefined}
      workerURL={WorkerURL}
      triggers={TRIGGERS_PROVIDER_PROPS}
      haul={{ useState: useHaulState }}
      color={{ useState: useColorContextState }}
      alamos={{ level: "info" }}
    >
      <Code.Provider importExtensions={monacoExtensions} initServices={monacoServices}>
        <ArcLSPClientSetter>
          <Vis.Canvas>
            <Layout.Window />
          </Vis.Canvas>
        </ArcLSPClientSetter>
      </Code.Provider>
    </Pluto.Provider>
  );
};

export const Console = (): ReactElement => (
  <Errors.OverlayWithoutStore>
    <Provider store={store}>
      <Errors.OverlayWithStore>
        <Layout.RendererProvider value={LAYOUT_RENDERERS}>
          <Layout.ContextMenuProvider value={CONTEXT_MENU_RENDERERS}>
            <Ontology.ServicesProvider services={SERVICES}>
              <Palette.CommandProvider
                commands={COMMANDS}
                fileIngestors={FILE_INGESTORS}
                extractors={EXTRACTORS}
              >
                <MainUnderContext />
              </Palette.CommandProvider>
            </Ontology.ServicesProvider>
          </Layout.ContextMenuProvider>
        </Layout.RendererProvider>
      </Errors.OverlayWithStore>
    </Provider>
  </Errors.OverlayWithoutStore>
);
