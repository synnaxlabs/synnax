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
import "@synnaxlabs/pluto/dist/pluto.css";

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
import { breaker, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Access } from "@/access";
import { Arc } from "@/arc";
import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Code } from "@/code";
import { Arc as ArcCode } from "@/code/arc";
import { COMMANDS } from "@/commands";
import { CSV } from "@/csv";
import { Docs } from "@/docs";
import { Errors } from "@/errors";
import { Export } from "@/export";
import { EXTRACTORS } from "@/extractors";
import { Hardware } from "@/hardware";
import { Import } from "@/import";
import { FILE_INGESTERS } from "@/ingesters";
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

const LSP_BREAKER_CONFIG: breaker.Config = {
  baseInterval: TimeSpan.seconds(1),
  maxRetries: 50,
  scale: 1.5,
};

const ArcLSPClientSetter = ({ children }: { children: ReactElement }): ReactElement => {
  const client = Synnax.use();
  const monaco = Code.useMonaco();
  useEffect(() => {
    if (monaco == null || client == null) return;
    const abortController = new AbortController();
    const { signal } = abortController;
    const abortPromise = new Promise<void>((r) =>
      signal.addEventListener("abort", () => r()),
    );
    let currentHandle: ArcCode.LSPClientHandle | null = null;
    let currentStream: ArcCode.LSPStream | null = null;
    const run = async () => {
      const b = new breaker.Breaker(LSP_BREAKER_CONFIG);
      while (!signal.aborted) {
        try {
          const stream = await client.arcs.openLSP();
          if (signal.aborted) {
            stream.closeSend();
            return;
          }
          currentStream = stream;
          const handle = await ArcCode.startLSPClient(stream);
          if (signal.aborted) {
            await ArcCode.stopLSPClient(handle.client);
            ArcCode.closeLSPStream(stream);
            return;
          }
          currentHandle = handle;
          b.reset();
          await Promise.race([handle.closed, abortPromise]);
          currentHandle = null;
          currentStream = null;
          await ArcCode.stopLSPClient(handle.client);
          ArcCode.closeLSPStream(stream);
          if (signal.aborted) return;
        } catch (e) {
          console.error("Arc LSP connection failed:", e);
          currentHandle = null;
          currentStream = null;
        }
        if (!(await b.wait())) {
          console.error("Arc LSP breaker exhausted, giving up reconnection");
          return;
        }
      }
    };
    run().catch(console.error);
    return () => {
      abortController.abort();
      if (currentHandle != null)
        ArcCode.stopLSPClient(currentHandle.client).catch(console.error);
      if (currentStream != null) ArcCode.closeLSPStream(currentStream);
    };
  }, [client, monaco]);
  return children;
};

const MONACO_SERVICES = [...ArcCode.SERVICES];

const MainUnderContext = (): ReactElement => {
  const theme = Layout.useThemeProvider();
  const cluster = Cluster.useSelect();
  useBlockDefaultDropBehavior();

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
      <Code.Provider initServices={MONACO_SERVICES}>
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
            <Import.FileIngestersProvider fileIngesters={FILE_INGESTERS}>
              <Export.ExtractorsProvider extractors={EXTRACTORS}>
                <Ontology.ServicesProvider services={SERVICES}>
                  <Palette.CommandProvider commands={COMMANDS}>
                    <MainUnderContext />
                  </Palette.CommandProvider>
                </Ontology.ServicesProvider>
              </Export.ExtractorsProvider>
            </Import.FileIngestersProvider>
          </Layout.ContextMenuProvider>
        </Layout.RendererProvider>
      </Errors.OverlayWithStore>
    </Provider>
  </Errors.OverlayWithoutStore>
);
