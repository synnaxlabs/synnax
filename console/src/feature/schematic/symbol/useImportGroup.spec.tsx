// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type ontology } from "@synnaxlabs/client";
import { Haul, type Status } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  childNames,
  client,
  createLegacySymbolFile,
  createSymbolPayload,
} from "@/feature/schematic/testutil";
import {
  createJSONFile,
  fakeDirectoryEntry,
  fakeFileEntry,
  FileDragSource,
  fireFileDrop,
  startFileDrag,
} from "@/platform/import/testutil";
import { Modals } from "@/platform/modals";
import {
  CaptureStatuses,
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  renderSuspended,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

const ZONE_TEXT = "Drop a .zip or folder here";

const renderModal = async () => {
  const statuses: Status.NotificationSpec[] = [];
  const { wrapper } = await createConsoleWrapper({ client });
  const Opener = (): ReactElement => {
    const open = Schematic.Symbol.useImportGroup();
    return <button onClick={() => open()}>open import</button>;
  };
  await renderSuspended(
    <Haul.Provider>
      <FileDragSource />
      <Opener />
      <Modals.Stack />
      <CaptureStatuses
        onStatuses={(next) => statuses.splice(0, statuses.length, ...next)}
      />
    </Haul.Provider>,
    { wrapper },
  );
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "open import" }));
  });
  await screen.findByText(ZONE_TEXT);
  return { statuses };
};

const successMessage = (name: string): string => `Imported symbol group "${name}"`;

const awaitStatus = async (
  statuses: Status.NotificationSpec[],
  variant: string,
  message: string,
): Promise<void> => {
  await waitFor(() =>
    expect(
      statuses.some((st) => st.variant === variant && st.message === message),
    ).toBe(true),
  );
};

/** Returns the permanent symbol group's child groups with the given name. */
const importedGroupsNamed = async (name: string): Promise<ontology.Resource[]> => {
  const root = await client.schematics.symbols.retrieveGroup();
  const groups = await client.ontology.children.retrieve({
    ids: group.ontologyID(root.key),
  });
  return groups.filter((g) => g.name === name);
};

/**
 * Creates a group with one symbol on the cluster and exports it, returning the
 * group's name, the exported symbol's name, and the bundle's zip bytes.
 */
const createExportedBundle = async () => {
  const root = await client.schematics.symbols.retrieveGroup();
  const grp = await client.groups.create({
    parent: group.ontologyID(root.key),
    name: uniqueName("zip_grp"),
  });
  const payload = createSymbolPayload(uniqueName("sym"));
  await client.schematics.symbols.create({
    ...payload,
    parent: group.ontologyID(grp.key),
  });
  const stream = await client.schematics.symbols.exportGroup(grp.key, {
    encoding: "JSON",
  });
  const bytes = new Uint8Array(await new Response(stream).arrayBuffer());
  return { groupName: grp.name, sourceKey: grp.key, symbolName: payload.name, bytes };
};

describe("Schematic.Symbol.useImportGroup", () => {
  describe("zip file", () => {
    it("imports a zip export picked from the file browser", async () => {
      const { groupName, bytes } = await createExportedBundle();
      const picker = interceptFilePicker();
      const { statuses } = await renderModal();
      fireEvent.click(screen.getByText(ZONE_TEXT));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([fakePickedFile(`${groupName}.zip`, bytes)]);
      await awaitStatus(statuses, "success", successMessage(groupName));
      // The export's source group shares the name, so the import adds a second.
      const groups = await importedGroupsNamed(groupName);
      expect(groups).toHaveLength(2);
      expect(screen.queryByText(ZONE_TEXT)).toBeNull();
    });

    it("imports a dropped zip export", async () => {
      const { groupName, sourceKey, symbolName, bytes } = await createExportedBundle();
      const { statuses } = await renderModal();
      startFileDrag();
      fireFileDrop(screen.getByText(ZONE_TEXT), [
        fakeFileEntry(new File([bytes], `${groupName}.zip`)),
      ]);
      await awaitStatus(statuses, "success", successMessage(groupName));
      const groups = await importedGroupsNamed(groupName);
      expect(groups).toHaveLength(2);
      const imported = groups.find((g) => g.id.key !== sourceKey);
      if (imported == null) throw new Error("imported group not found");
      expect(await childNames(imported.id)).toContain(symbolName);
    });

    it("rejects a dropped file that is not a zip", async () => {
      const { statuses } = await renderModal();
      startFileDrag();
      fireFileDrop(screen.getByText(ZONE_TEXT), [
        fakeFileEntry(createJSONFile("manifest.json", { version: 2 })),
      ]);
      await awaitStatus(statuses, "error", "Failed to import symbol group");
      expect(screen.queryByText(ZONE_TEXT)).not.toBeNull();
    });
  });

  describe("directory", () => {
    it("imports a dropped folder", async () => {
      const groupName = uniqueName("dropped_grp");
      const symbolName = uniqueName("sym");
      const { statuses } = await renderModal();
      startFileDrag();
      fireFileDrop(screen.getByText(ZONE_TEXT), [
        fakeDirectoryEntry(groupName, [
          createJSONFile("manifest.json", {
            version: 2,
            type: "symbol_group",
            name: groupName,
          }),
          createJSONFile(
            `${symbolName}.json`,
            JSON.parse(createLegacySymbolFile(symbolName)),
          ),
        ]),
      ]);
      await awaitStatus(statuses, "success", successMessage(groupName));
      const [imported] = await importedGroupsNamed(groupName);
      if (imported == null) throw new Error("imported group not found");
      expect(await childNames(imported.id)).toContain(symbolName);
    });

    it("imports every symbol a legacy manifest declares", async () => {
      const picker = interceptFilePicker();
      const { statuses } = await renderModal();
      const groupName = uniqueName("imported_grp");
      const symbolName = uniqueName("sym");
      const manifest = {
        version: 1,
        type: "symbol_group",
        name: groupName,
        symbols: [{ file: "sym1.json", key: "k1", name: symbolName }],
      };
      fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([
        fakePickedFile(
          "manifest.json",
          JSON.stringify(manifest),
          `${groupName}/manifest.json`,
        ),
        fakePickedFile(
          "sym1.json",
          createLegacySymbolFile(symbolName),
          `${groupName}/sym1.json`,
        ),
      ]);
      await awaitStatus(statuses, "success", successMessage(groupName));
      const [imported] = await importedGroupsNamed(groupName);
      if (imported == null) throw new Error("imported group not found");
      expect(await childNames(imported.id)).toContain(symbolName);
    });

    it("raises an error when the picked folder has no manifest", async () => {
      const picker = interceptFilePicker();
      const { statuses } = await renderModal();
      fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([
        fakePickedFile("stray.json", "{}", "no_manifest_dir/stray.json"),
      ]);
      await awaitStatus(statuses, "error", "Failed to import symbol group");
      expect(screen.queryByText(ZONE_TEXT)).not.toBeNull();
    });

    it("imports nothing when the manifest declares a missing member", async () => {
      const picker = interceptFilePicker();
      const { statuses } = await renderModal();
      const groupName = uniqueName("partial_grp");
      const manifest = {
        version: 1,
        type: "symbol_group",
        name: groupName,
        symbols: [
          { file: "ok.json", key: "k1", name: "ok" },
          { file: "missing.json", key: "k2", name: "missing" },
        ],
      };
      fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([
        fakePickedFile(
          "manifest.json",
          JSON.stringify(manifest),
          `${groupName}/manifest.json`,
        ),
        fakePickedFile("ok.json", createLegacySymbolFile("ok"), `${groupName}/ok.json`),
      ]);
      await awaitStatus(statuses, "error", "Failed to import symbol group");
      expect(await importedGroupsNamed(groupName)).toHaveLength(0);
    });

    it("infers membership from the directory for a Core-written manifest", async () => {
      const picker = interceptFilePicker();
      const { statuses } = await renderModal();
      const groupName = uniqueName("core_grp");
      const symbolName = uniqueName("sym");
      const manifest = { version: 2, type: "symbol_group", name: groupName };
      fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([
        fakePickedFile(
          "manifest.json",
          JSON.stringify(manifest),
          `${groupName}/manifest.json`,
        ),
        fakePickedFile(
          `${symbolName}.json`,
          createLegacySymbolFile(symbolName),
          `${groupName}/${symbolName}.json`,
        ),
      ]);
      await awaitStatus(statuses, "success", successMessage(groupName));
      const [imported] = await importedGroupsNamed(groupName);
      if (imported == null) throw new Error("imported group not found");
      expect(await childNames(imported.id)).toContain(symbolName);
    });
  });
});
