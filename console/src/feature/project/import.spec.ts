// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, project, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Access } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { assert, describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { type Import } from "@/platform/import";
import { Session } from "@/session";
import { assertDefined, createConsoleWrapper, type TestStore } from "@/testutil";

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

// A legacy (layout-slice era) export tiling file for the two component files.
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
  granted: boolean;
}

const selectImportedProject = (store: TestStore): project.Key => {
  const key = Session.Project.selectOptionalSelected(store.getState());
  assertDefined(key, "no project selected after import");
  return key;
};

const retrieveProjectChildren = async (
  key: project.Key,
): Promise<ontology.Resource[]> =>
  await client.ontology.children.retrieve({ ids: project.ontologyID(key) });

describe("project import", () => {
  const runImport = async (fileList: Import.File[]): Promise<TestStore> => {
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook<HarnessValue, unknown>(
      () => ({ granted: Access.useUpdateGranted(project.TYPE_ONTOLOGY_ID) }),
      { wrapper },
    );
    await waitFor(() => expect(result.current.granted).toBe(true));
    await act(async () => {
      await Project.ingest(`proj-${id.create()}`, fileList, { client, store });
    });
    return store;
  };

  const componentFiles = (): Import.File[] => [
    { name: "Operator.json", data: SCHEMATIC_DATA },
    { name: "Thermocouples.json", data: TABLE_DATA },
  ];

  describe("manifest bundles", () => {
    const bundleFiles = (name: string): Import.File[] => [
      {
        name: Project.MANIFEST_FILE_NAME,
        path: Project.MANIFEST_FILE_NAME,
        data: { version: 1, type: "project", name },
      },
      { name: "Operator.json", path: "Propulsion/Operator.json", data: SCHEMATIC_DATA },
      {
        name: "Main.json",
        path: "Main.json",
        data: {
          version: 0,
          type: "panel",
          name: "Main",
          root: {
            variant: "leaf",
            tabs: [
              {
                key: uuid.create(),
                variant: "resource",
                resource: "Propulsion/Operator.json",
              },
            ],
          },
        },
      },
    ];

    it("recreates groups and resolves path references", async () => {
      const name = `bundle-${id.create()}`;
      const store = await runImport(bundleFiles(name));
      const projectKey = selectImportedProject(store);
      const imported = await client.projects.retrieve(projectKey);
      expect(imported.name).toBe(name);
      const children = await retrieveProjectChildren(projectKey);
      const grp = children.find(({ id }) => id.type === "group");
      assertDefined(grp, "no group created for the bundle directory");
      expect(grp.name).toBe("Propulsion");
      const grouped = await client.ontology.children.retrieve({ ids: grp.id });
      expect(grouped.map(({ id }) => id.type)).toEqual([SCHEMATIC_TYPE]);
      const panelChild = children.find(({ id }) => id.type === "panel");
      assertDefined(panelChild, "no panel created from the bundle");
      const [imported2] = await client.panels.retrieve({ keys: [panelChild.id.key] });
      assert(imported2.root.variant === "leaf", "expected a leaf root");
      const [tab] = imported2.root.tabs;
      assert(tab.variant === "resource", "expected a resource tab");
      expect(tab.resource).toEqual(grouped[0].id);
    });

    it("imports a root LAYOUT.json member when a manifest is present", async () => {
      const files = bundleFiles(`bundle-${id.create()}`);
      files.push({
        name: Project.LAYOUT_FILE_NAME,
        path: Project.LAYOUT_FILE_NAME,
        data: { ...SCHEMATIC_DATA, key: uuid.create(), name: "LAYOUT" },
      });
      const store = await runImport(files);
      const projectKey = selectImportedProject(store);
      const children = await retrieveProjectChildren(projectKey);
      const member = children.find(({ name }) => name === "LAYOUT");
      assertDefined(member, "LAYOUT.json member was not imported");
      expect(member.id.type).toBe(SCHEMATIC_TYPE);
    });

    it("rejects a bundle of another kind", async () => {
      const files = bundleFiles(`bundle-${id.create()}`);
      files[0] = { ...files[0], data: { version: 1, type: "symbol_group", name: "g" } };
      await expect(runImport(files)).rejects.toThrow(
        "Cannot import a symbol_group bundle",
      );
    });

    it("errors when a panel references a file that is not in the bundle", async () => {
      const files = bundleFiles(`bundle-${id.create()}`).filter(
        ({ path }) => path !== "Propulsion/Operator.json",
      );
      await expect(runImport(files)).rejects.toThrow("Propulsion/Operator.json");
    });
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
