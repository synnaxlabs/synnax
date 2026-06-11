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
  type Alamos,
  type Color,
  type Haul,
  Pluto,
  preventDefault,
  type state,
  type Triggers,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { Access } from "@/access";
import { Arc } from "@/arc";
import { Channel } from "@/channel";
import { Cluster } from "@/cluster";
import { Code } from "@/code";
import { COMMANDS } from "@/commands";
import { CSV } from "@/csv";
import { Docs } from "@/docs";
import { Errors } from "@/errors";
import { Export } from "@/export";
import { EXTRACTORS } from "@/extractors";
import { Framer } from "@/framer";
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
import { Project } from "@/project";
import { Range } from "@/range";
import { Runtime } from "@/runtime";
import { Schematic } from "@/schematic";
import { Selector } from "@/selector";
import { SERVICES } from "@/services";
import { Status } from "@/status";
import { store } from "@/store";
import { Table } from "@/table";
import { User } from "@/user";
import { Version } from "@/version";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";

const LAYOUT_RENDERERS: Record<string, Layout.Renderer> = {
  ...Channel.LAYOUTS,
  ...Cluster.LAYOUTS,
  ...CSV.LAYOUTS,
  ...Framer.LAYOUTS,
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
  ...Project.LAYOUTS,
  ...Arc.LAYOUTS,
  ...Status.LAYOUTS,
  ...Access.LAYOUTS,
};

// CONTEXT_MENU_RENDERERS lets a content type replace its panel tab's context
// menu wholesale (Panel.ContextMenu falls back to the standard tab items). No
// type currently registers one; the legacy schematic/lineplot entries were
// trivial wrappers around the shared default items.
const CONTEXT_MENU_RENDERERS: Record<string, Layout.ContextMenuRenderer> = {};

// SELECTABLES is the app-wide "create a component" registry, served to the panel
// empty-tab selector and the create affordances through Selector.Provider.
const SELECTABLES: Selector.Selectable[] = [
  ...LinePlot.SELECTABLES,
  ...Schematic.SELECTABLES,
  ...Log.SELECTABLES,
  ...Table.SELECTABLES,
  ...Hardware.SELECTABLES,
  ...Arc.SELECTABLES,
];

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

const MONACO_SERVICES = Arc.LSP.SERVICES;

const ALAMOS_PROPS: Alamos.ProviderProps = { level: "info" };

const HAUL_PROPS: Haul.ProviderProps = { useState: useHaulState };
const COLOR_PROPS: Color.ProviderProps = { useState: useColorContextState };

const MainUnderContext = (): ReactElement => {
  const theme = Layout.useThemeProvider();
  const cluster = Cluster.useSelect();
  useBlockDefaultDropBehavior();
  Runtime.useExternalLinkHandler();

  return (
    <Pluto.Provider
      theming={theme}
      workerEnabled
      connParams={cluster ?? undefined}
      workerURL={WorkerURL}
      triggers={TRIGGERS_PROVIDER_PROPS}
      haul={HAUL_PROPS}
      color={COLOR_PROPS}
      alamos={ALAMOS_PROPS}
    >
      <Code.Provider initServices={MONACO_SERVICES}>
        <Arc.LSP.Provider>
          <Vis.Canvas>
            <Layout.Window />
          </Vis.Canvas>
        </Arc.LSP.Provider>
      </Code.Provider>
    </Pluto.Provider>
  );
};

export const Console = (): ReactElement => (
  <Errors.OverlayWithoutStore>
    <Provider store={store}>
      <Errors.OverlayWithStore>
        <Layout.RendererProvider value={LAYOUT_RENDERERS}>
          <Selector.Provider value={SELECTABLES}>
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
          </Selector.Provider>
        </Layout.RendererProvider>
      </Errors.OverlayWithStore>
    </Provider>
  </Errors.OverlayWithoutStore>
);
