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

import { COMMANDS } from "@/app/commands";
import { EXTRACTORS } from "@/app/extractors";
import { FILE_INGESTERS } from "@/app/ingesters";
import { SERVICES } from "@/app/services";
import { SNAPSHOT_SERVICES } from "@/app/snapshots";
import { TABS } from "@/app/tabs";
import { Vis } from "@/app/vis";
import { Window } from "@/app/Window";
import { Task } from "@/feature/task";
import { Errors } from "@/platform/errors";
import { Export } from "@/platform/export";
import { Import } from "@/platform/import";
import { Ontology } from "@/platform/ontology";
import { Palette } from "@/platform/palette";
import { Panel } from "@/platform/panel";
import { Range as PlatformRange } from "@/platform/range";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";
import WorkerURL from "@/worker?worker&url";

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
  const hauled = Session.Haul.useSelectHauling();
  const dispatch = Session.useDispatch();
  const onHauledChange = useCallback(
    (state: Haul.DraggingState) => dispatch(Session.Haul.setHauled(state)),
    [dispatch],
  );
  return [hauled, onHauledChange];
};

const useColorContextState: state.PureUse<Color.ContextState> = () => {
  const colorContext = Session.Color.useSelectContext();
  const dispatch = Session.useDispatch();
  const onColorContextChange = useCallback(
    (state: Color.ContextState) => dispatch(Session.Color.setContext(state)),
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

const AppUnderContext = (): ReactElement => {
  const cluster = Session.Cluster.useSelectState();
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
          <Window />
        </Session.Modals.Provider>
      </Vis.Canvas>
    </Pluto.Provider>
  );
};

export const App = (): ReactElement => {
  const storeRef = useInitializerRef(() => Session.configureStore());
  return (
    <Errors.OverlayWithoutStore>
      <Provider store={storeRef.current}>
        <Errors.OverlayWithStore>
          <Panel.RendererContext value={TABS}>
            <Import.FileIngestersProvider fileIngesters={FILE_INGESTERS}>
              <Export.ExtractorsProvider extractors={EXTRACTORS}>
                <Ontology.ServicesProvider services={SERVICES}>
                  <Palette.CommandProvider commands={COMMANDS}>
                    <PlatformRange.SnapshotServicesProvider services={SNAPSHOT_SERVICES}>
                      <Task.RegistryProvider registry={Task.REGISTRY}>
                        <AppUnderContext />
                      </Task.RegistryProvider>
                    </PlatformRange.SnapshotServicesProvider>
                  </Palette.CommandProvider>
                </Ontology.ServicesProvider>
              </Export.ExtractorsProvider>
            </Import.FileIngestersProvider>
          </Panel.RendererContext>
        </Errors.OverlayWithStore>
      </Provider>
    </Errors.OverlayWithoutStore>
  );
};
