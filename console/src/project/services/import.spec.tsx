// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  combineReducers,
  configureStore,
  type Reducer,
  type Store,
  type UnknownAction,
} from "@reduxjs/toolkit";
import { createTestClient, project, type Synnax } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Access, Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { deep, id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeAll, describe, expect, it } from "vitest";

import { type Import } from "@/import";
import { Schematic } from "@/layered/service/schematic";
import { Session } from "@/layered/session";
import { Layout } from "@/layout";
import { Project } from "@/project";
import { ProjectServices } from "@/project/services";
import { Table } from "@/table";
import { TableServices } from "@/table/services";

const client: Synnax = createTestClient();

const WINDOW_KEY = "main";
const SCHEMATIC_TYPE = "schematic";
const TABLE_TYPE = "table";
const OPERATOR_KEY = "34c0a87c-3f72-42d2-8cac-75bc1e2631b1";
const THERMO_KEY = "cdb27884-a73f-4696-bcee-a71c1f6625bd";

// The real ingesters for these types; the full FILE_INGESTERS registry would drag in
// the Arc/Monaco editor, which Vitest can't load.
const FILE_INGESTERS: Import.FileIngesters = {
  ...Schematic.ImEx.FILE_INGESTERS,
  ...TableServices.FILE_INGESTERS,
};

const SCHEMATIC_DATA = {
  key: OPERATOR_KEY,
  name: "Operator",
  type: SCHEMATIC_TYPE,
  version: "6.0.0",
  snapshot: false,
  nodes: [{ key: "n1", position: { x: 0, y: 0 } }],
  edges: [],
  configs: { n1: { variant: "valve", color: [28, 28, 28, 1] } },
};

const TABLE_DATA = {
  key: THERMO_KEY,
  name: "Thermocouples",
  type: TABLE_TYPE,
  version: "1.0.0",
  rows: [{ size: 36, cells: ["c1"] }],
  columns: [{ size: 72 }],
  cells: { c1: { key: "c1", variant: "text", props: { value: "hello" } } },
};

const rootReducer = combineReducers({
  [Layout.SLICE_NAME]: Layout.reducer,
  [Session.Schematic.SLICE_NAME]: Session.Schematic.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Project.SLICE_NAME]: Project.reducer,
  drift: Drift.reducer,
}) as unknown as Reducer<Record<string, unknown>, UnknownAction>;

// An exported layout slice with a schematic and a table tab, each keyed to match the
// resource key in the corresponding component file.
const exportedSlice = (): Layout.SliceState => {
  let s = Layout.reducer(undefined, { type: "@@INIT" });
  s = Layout.reducer(
    s,
    Layout.place({
      windowKey: WINDOW_KEY,
      key: OPERATOR_KEY,
      type: SCHEMATIC_TYPE,
      name: "Operator",
      location: "mosaic",
    }),
  );
  s = Layout.reducer(
    s,
    Layout.place({
      windowKey: WINDOW_KEY,
      key: THERMO_KEY,
      type: TABLE_TYPE,
      name: "Thermocouples",
      location: "mosaic",
    }),
  );
  return s;
};

const layoutsOfType = (store: Store, type: string): Layout.State[] =>
  Object.values(Layout.selectSliceState(store.getState() as never).layouts).filter(
    (l) => l.type === type,
  );

interface HarnessValue {
  placer: Layout.Placer;
  fluxStore: Pluto.FluxStore;
  granted: boolean;
}

describe("project import", () => {
  let fluxClient: Flux.Client;

  beforeAll(async () => {
    fluxClient = new Flux.Client({
      client,
      storeConfig: Pluto.FLUX_STORE_CONFIG,
      handleError: () => {},
      handleAsyncError: async () => {},
    });
    await fluxClient.awaitInitialized();
  });

  const runImport = async (fileList: Import.File[] = files()): Promise<Store> => {
    const store = configureStore({ reducer: rootReducer });
    const wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Provider store={store}>
        <Status.Aggregator>
          <PSynnax.TestProvider client={client}>
            <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
          </PSynnax.TestProvider>
        </Status.Aggregator>
      </Provider>
    );
    const { result } = renderHook<HarnessValue, unknown>(
      () => ({
        placer: Layout.usePlacer(),
        fluxStore: Flux.useStore<Pluto.FluxStore>(),
        granted: Access.useUpdateGranted(project.TYPE_ONTOLOGY_ID),
      }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.granted).toBe(true));
    await act(async () => {
      await ProjectServices.ingest(`proj-${id.create()}`, fileList, {
        client,
        fileIngesters: FILE_INGESTERS,
        placeLayout: result.current.placer,
        store,
        fluxStore: result.current.fluxStore,
      });
    });
    return store;
  };

  const files = (layoutSlice: unknown = exportedSlice()): Import.File[] => [
    { name: Project.LAYOUT_FILE_NAME, data: layoutSlice },
    { name: "Operator.json", data: SCHEMATIC_DATA },
    { name: "Thermocouples.json", data: TABLE_DATA },
  ];

  // An exported slice whose themes predate newer color fields; anySliceStateZ would
  // reject them outright.
  const staleThemesSlice = (): unknown => {
    const slice = deep.copy(exportedSlice()) as unknown as {
      themes: Record<string, { colors: Record<string, unknown> }>;
    };
    Object.values(slice.themes).forEach(({ colors }) => {
      delete colors.primaryText;
      delete colors.errorText;
      delete colors.warningText;
    });
    return slice;
  };

  it("places exactly one tab per imported schematic and table", async () => {
    const store = await runImport();
    expect(layoutsOfType(store, SCHEMATIC_TYPE)).toHaveLength(1);
    expect(layoutsOfType(store, TABLE_TYPE)).toHaveLength(1);
  });

  it("links every imported tab to a resource that exists in the cluster", async () => {
    const store = await runImport();
    const [schematicLayout] = layoutsOfType(store, SCHEMATIC_TYPE);
    const [tableLayout] = layoutsOfType(store, TABLE_TYPE);
    await expect(
      client.schematics.retrieve({ key: schematicLayout.key }),
    ).resolves.toBeDefined();
    await expect(
      client.tables.retrieve({ key: tableLayout.key }),
    ).resolves.toBeDefined();
  });

  it("imports a project whose exported themes predate current theme fields", async () => {
    const store = await runImport(files(staleThemesSlice()));
    expect(layoutsOfType(store, SCHEMATIC_TYPE)).toHaveLength(1);
    expect(layoutsOfType(store, TABLE_TYPE)).toHaveLength(1);
  });
});
