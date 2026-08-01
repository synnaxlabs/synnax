// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ontology,
  panel,
  project,
  schematic,
  type Synnax,
  table,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Access, Flux, type Pluto } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { type Import } from "@/platform/import";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore } from "@/testutil";

const client: Synnax = createTestClient();

const SCHEMATIC_TYPE = "schematic";
const TABLE_TYPE = "table";
const OPERATOR_KEY = "34c0a87c-3f72-42d2-8cac-75bc1e2631b1";
const THERMO_KEY = "cdb27884-a73f-4696-bcee-a71c1f6625bd";

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

// An exported panel with a schematic and a table tab, each referencing the resource
// key in the corresponding component file.
const exportedPanels = (): panel.Panel[] => [
  panel.panelZ.parse({
    name: "Main",
    root: {
      variant: "leaf",
      tabs: [
        {
          variant: "resource",
          key: uuid.create(),
          resource: schematic.ontologyID(OPERATOR_KEY),
        },
        {
          variant: "resource",
          key: uuid.create(),
          resource: table.ontologyID(THERMO_KEY),
        },
      ],
    },
  }),
];

// A legacy (layout-slice era) export tiling file for the same two components.
const legacyLayoutSlice = (): unknown => ({
  layouts: {
    [OPERATOR_KEY]: {
      key: OPERATOR_KEY,
      windowKey: "main",
      type: SCHEMATIC_TYPE,
      name: "Operator",
      location: "mosaic",
    },
    [THERMO_KEY]: {
      key: THERMO_KEY,
      windowKey: "main",
      type: TABLE_TYPE,
      name: "Thermocouples",
      location: "mosaic",
    },
  },
});

interface HarnessValue {
  openTab: Panel.OpenTab;
  fluxStore: Pluto.FluxStore;
  granted: boolean;
}

const selectImportedProject = (store: TestStore): project.Key => {
  const key = Session.Project.selectOptionalSelected(store.getState());
  if (key == null) throw new Error("no project selected after import");
  return key;
};

const retrieveProjectChildren = async (
  key: project.Key,
): Promise<ontology.Resource[]> =>
  await client.ontology.retrieveChildren(project.ontologyID(key));

describe("project import", () => {
  const runImport = async (fileList: Import.File[]): Promise<TestStore> => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook<HarnessValue, unknown>(
      () => ({
        openTab: Panel.useOpenTab(),
        fluxStore: Flux.useStore<Pluto.FluxStore>(),
        granted: Access.useUpdateGranted(project.TYPE_ONTOLOGY_ID),
      }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.granted).toBe(true));
    await act(async () => {
      // No client-side ingesters: visualization files route through the server.
      await Project.ingest(`proj-${id.create()}`, fileList, {
        client,
        fileIngesters: {},
        openTab: result.current.openTab,
        store,
        fluxStore: result.current.fluxStore,
      });
    });
    return store;
  };

  const componentFiles = (): Import.File[] => [
    { name: "Operator.json", data: SCHEMATIC_DATA },
    { name: "Thermocouples.json", data: TABLE_DATA },
  ];

  const files = (): Import.File[] => [
    { name: Project.PANELS_FILE_NAME, data: exportedPanels() },
    ...componentFiles(),
  ];

  it("creates the project's panels with tabs pointing at the created resources", async () => {
    const store = await runImport(files());
    const projectKey = selectImportedProject(store);
    const children = await retrieveProjectChildren(projectKey);
    const panelKeys = children
      .filter(({ id }) => id.type === "panel")
      .map(({ id }) => id.key);
    expect(panelKeys).toHaveLength(1);
    const [imported] = await client.panels.retrieve(panelKeys);
    expect(imported.name).toBe("Main");
    if (imported.root.variant !== "leaf") throw new Error("expected a leaf root");
    const resources = imported.root.tabs.map((tab) => {
      if (tab.variant !== "resource") throw new Error("expected resource tabs");
      return tab.resource;
    });
    expect(resources.map(({ type }) => type)).toEqual([SCHEMATIC_TYPE, TABLE_TYPE]);
    const [schematicID, tableID] = resources;
    const importedSchematic = await client.schematics.retrieve({
      key: schematicID.key,
    });
    expect(importedSchematic.name).toBe("Operator");
    const importedTable = await client.tables.retrieve({ key: tableID.key });
    expect(importedTable.name).toBe("Thermocouples");
  });

  it("recreates the visualization documents of a legacy export without panels", async () => {
    const store = await runImport([
      { name: Project.LAYOUT_FILE_NAME, data: legacyLayoutSlice() },
      ...componentFiles(),
    ]);
    const projectKey = selectImportedProject(store);
    const children = await retrieveProjectChildren(projectKey);
    const types = children.map(({ id }) => id.type);
    expect(types).toContain(SCHEMATIC_TYPE);
    expect(types).toContain(TABLE_TYPE);
    // Legacy mosaic tilings are dropped on import, so no panels are created.
    expect(types).not.toContain("panel");
  });
});
