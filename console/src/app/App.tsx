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
  useInitializerRef,
} from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect } from "react";
import { useDispatch } from "react-redux";

import { COMMANDS } from "@/app/commands";
import { Cluster } from "@/cluster";
import { Errors } from "@/component/errors";
import { Palette } from "@/component/palette";
import { Export } from "@/export";
import { EXTRACTORS } from "@/extractors";
import { Hardware } from "@/hardware";
import { Import } from "@/import";
import { FILE_INGESTERS } from "@/ingesters";
import { Layout } from "@/layout";
import { Layouts } from "@/layouts";
import { Range } from "@/range";
import { Runtime } from "@/runtime";
import { Arc } from "@/service/arc";
import { Docs } from "@/service/docs";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Ontology } from "@/service/ontology";
import { Schematic } from "@/service/schematic";
import { Status } from "@/service/status";
import { Table } from "@/service/table";
import { SERVICES } from "@/services";
import { Session } from "@/session";
import { store } from "@/session/store";
import { Vis } from "@/vis";
import WorkerURL from "@/worker?worker&url";

const LAYOUT_RENDERERS: Record<string, Layout.Renderer> = {
  ...Docs.LAYOUTS,
  ...Hardware.LAYOUTS,
  ...Layouts.LAYOUTS,
  ...LinePlot.LAYOUTS,
  ...Log.LAYOUTS,
  ...Range.LAYOUTS,
  ...Schematic.LAYOUTS,
  ...Table.LAYOUTS,
  ...Vis.LAYOUTS,
  ...Arc.LAYOUTS,
  ...Status.LAYOUTS,
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
  const dispatch = Session.useDispatch();
  const onHauledChange = useCallback(
    (state: Haul.DraggingState) => dispatch(Layout.setHauled(state)),
    [dispatch],
  );
  return [hauled, onHauledChange];
};

const useColorContextState: state.PureUse<Color.ContextState> = () => {
  const colorContext = Layout.useSelectColorContext();
  const dispatch = Session.useDispatch();
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

const ALAMOS_PROPS: Alamos.ProviderProps = { level: "info" };

const HAUL_PROPS: Haul.ProviderProps = { useState: useHaulState };
const COLOR_PROPS: Color.ProviderProps = { useState: useColorContextState };

const MainUnderContext = (): ReactElement => {
  const cluster = Cluster.useSelect();
  const themingProps = Session.Theme.useProviderProps();
  useBlockDefaultDropBehavior();
  Runtime.useExternalLinkHandler();

  return (
    <Pluto.Provider
      workerEnabled
      connParams={cluster ?? undefined}
      workerURL={WorkerURL}
      triggers={TRIGGERS_PROVIDER_PROPS}
      haul={HAUL_PROPS}
      color={COLOR_PROPS}
      alamos={ALAMOS_PROPS}
      theming={themingProps}
    >
      <Vis.Canvas>
        <Session.Modals.Provider>
          <Layout.Window />
        </Session.Modals.Provider>
      </Vis.Canvas>
    </Pluto.Provider>
  );
};

export const Console = (): ReactElement => {
  const storeRef = useInitializerRef(() => Session.createStore());
  return (
    <Errors.OverlayWithoutStore>
      <Provider store={storeRef.current}>
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
};
