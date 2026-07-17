// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, type ontology } from "@synnaxlabs/client";
import { theming } from "@synnaxlabs/pluto/ether";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  client,
  createSymbolPayload,
  getEditSectionHeaderButton,
  getOverlayHandles,
  getPreviewTransformWrapper,
  SYMBOL_FILE_DROP_PROMPT,
} from "@/feature/schematic/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import {
  fakePickedFile,
  findDialogTrigger,
  getIconButton,
  interceptFilePicker,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

// jsdom's XML parser does not attach CSSStyleDeclaration to painted SVG shapes, which
// normalizeSVG's computed-style pass requires; a shapeless SVG exercises the same
// production load path without touching that jsdom hole.
const SHAPELESS_SVG = '<svg viewBox="0 0 100 100"><g id="grp"></g></svg>';

const createParentGroup = async (): Promise<group.Group> => {
  const root = await client.schematics.symbols.retrieveGroup();
  return await client.groups.create({
    parent: group.ontologyID(root.key),
    name: uniqueName("edit_grp"),
  });
};

const childNames = async (id: ontology.ID): Promise<string[]> => {
  const children = await client.ontology.retrieveChildren(id);
  const symbols = await client.schematics.symbols.retrieve({
    keys: children.map((c) => c.id.key),
  });
  return symbols.map((s) => s.name);
};

const openCreateModal = async (createKey?: string) => {
  const grp = await createParentGroup();
  const picker = interceptFilePicker();
  const handle = await renderModalOpener(
    Schematic.Symbol.Edit.useModal,
    [{ parent: group.ontologyID(grp.key), createKey }],
    { client, additionalRegistry: theming.REGISTRY },
  );
  return { grp, picker, handle };
};

const loadSVG = async (
  picker: ReturnType<typeof interceptFilePicker>,
  fileName = "flow_meter.svg",
  contents = SHAPELESS_SVG,
) => {
  fireEvent.click(await screen.findByText(SYMBOL_FILE_DROP_PROMPT));
  await waitFor(() => expect(picker.lastInput()).toBeDefined());
  picker.selectFiles([fakePickedFile(fileName, contents)]);
  await screen.findByText("Colors");
};

const findNameInput = (): HTMLInputElement =>
  screen.getByPlaceholderText<HTMLInputElement>("Symbol Name");

describe("Schematic.Symbol.Edit.useModal", () => {
  describe("create mode", () => {
    it("loads a picked SVG, names the symbol after the file, and shows the sidebar", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      await waitFor(() => expect(findNameInput().value).toBe("Flow Meter"));
      expect(screen.getByText("States")).toBeDefined();
      expect(screen.getByText("Handles")).toBeDefined();
      expect(screen.getByText("Properties")).toBeDefined();
      expect(screen.getByText("Scale stroke")).toBeDefined();
    });

    it("creates the symbol under the parent group on save", async () => {
      const { grp, picker } = await openCreateModal();
      await loadSVG(picker);
      const name = uniqueName("created_symbol");
      fireEvent.change(findNameInput(), { target: { value: name } });
      fireEvent.click(findButton("Create"));
      await waitFor(async () =>
        expect(await childNames(group.ontologyID(grp.key))).toContain(name),
      );
      await waitFor(() =>
        expect(screen.queryByText(SYMBOL_FILE_DROP_PROMPT)).toBeNull(),
      );
    });

    it("persists the symbol under a forced key when createKey is provided", async () => {
      const createKey = uuid.create();
      const { picker } = await openCreateModal(createKey);
      await loadSVG(picker);
      const name = uniqueName("healed_symbol");
      fireEvent.change(findNameInput(), { target: { value: name } });
      fireEvent.click(findButton("Create"));
      await waitFor(async () => {
        const created = await client.schematics.symbols.retrieve({ key: createKey });
        expect(created.name).toBe(name);
      });
    });

    it("adds a new region to the selected state", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Colors"));
      expect(await screen.findByText("Region 1")).toBeDefined();
      expect(screen.getByText("0 Elements")).toBeDefined();
    });

    it("adds a placeable handle at the symbol center", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Handles"));
      expect(await screen.findByText("Handle 1")).toBeDefined();
      expect(screen.getByText("(50%, 50%)")).toBeDefined();
    });

    it("zooms the preview with keyboard shortcuts", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      await screen.findByText("100%");
      fireEvent.keyDown(window, { key: "=", ctrlKey: true });
      expect(await screen.findByText("120%")).toBeDefined();
      fireEvent.keyDown(window, { key: "0", ctrlKey: true });
      expect(await screen.findByText("100%")).toBeDefined();
      fireEvent.keyDown(window, { key: "-", ctrlKey: true });
      expect(await screen.findByText("83%")).toBeDefined();
    });

    it("pans the preview while shift-dragging", async () => {
      const { picker, handle } = await openCreateModal();
      await loadSVG(picker);
      await screen.findByText("100%");
      const wrapper = getPreviewTransformWrapper(handle.baseElement);
      const container = wrapper.parentElement;
      if (container == null) throw new Error("preview container not found");
      fireEvent.mouseDown(container, {
        button: 0,
        shiftKey: true,
        clientX: 100,
        clientY: 100,
      });
      fireEvent.mouseMove(container, { clientX: 120, clientY: 130 });
      await waitFor(() =>
        expect(wrapper.style.transform).toContain("translate(20px, 30px)"),
      );
      fireEvent.mouseUp(container);
      fireEvent.mouseMove(container, { clientX: 200, clientY: 200 });
      expect(wrapper.style.transform).toContain("translate(20px, 30px)");
    });

    it("zooms the preview with the scroll wheel", async () => {
      const { picker, handle } = await openCreateModal();
      await loadSVG(picker);
      await screen.findByText("100%");
      const wrapper = getPreviewTransformWrapper(handle.baseElement);
      const container = wrapper.parentElement;
      if (container == null) throw new Error("preview container not found");
      fireEvent.wheel(container, { deltaY: -1 });
      expect(await screen.findByText("110%")).toBeDefined();
      fireEvent.wheel(container, { deltaY: 1 });
      expect(await screen.findByText("99%")).toBeDefined();
    });

    it("selects a handle from the preview overlay and re-orients it", async () => {
      const { picker, handle } = await openCreateModal();
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Handles"));
      await screen.findByText("Handle 1");
      fireEvent.click(getEditSectionHeaderButton("Handles"));
      await screen.findByText("Handle 2");
      const overlayHandles = getOverlayHandles(handle.baseElement, "left");
      expect(overlayHandles).toHaveLength(2);
      expect(overlayHandles[1].className).toContain("console--selected");
      fireEvent.mouseDown(overlayHandles[0]);
      await waitFor(() =>
        expect(overlayHandles[0].className).toContain("console--selected"),
      );
      fireEvent.click(getIconButton(handle.baseElement, "arrow-right"));
      await waitFor(() =>
        expect(getOverlayHandles(handle.baseElement, "right")).toHaveLength(1),
      );
    });

    it("adds an active state when the variant switches to actuator", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      fireEvent.click(await findDialogTrigger());
      fireEvent.click(await screen.findByText("Actuator"));
      expect(await screen.findByText("Base")).toBeDefined();
      expect(screen.getByText("Active")).toBeDefined();
    });

    it("rejects a non-SVG file without opening the editor", async () => {
      const { picker } = await openCreateModal();
      fireEvent.click(await screen.findByText(SYMBOL_FILE_DROP_PROMPT));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([fakePickedFile("not_an_svg.txt", "hello")]);
      await waitFor(() => expect(screen.queryByText("Colors")).toBeNull());
      expect(screen.getByText(SYMBOL_FILE_DROP_PROMPT)).toBeDefined();
    });
  });

  describe("edit mode", () => {
    it("loads the existing symbol and renames it on save", async () => {
      const grp = await createParentGroup();
      const original = uniqueName("existing");
      const symbol = await client.schematics.symbols.create({
        ...createSymbolPayload(original),
        parent: group.ontologyID(grp.key),
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, parent: group.ontologyID(grp.key) }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      const nameInput = await waitFor(() => {
        const input = findNameInput();
        expect(input.value).toBe(original);
        return input;
      });
      const renamed = uniqueName("renamed_symbol");
      fireEvent.change(nameInput, { target: { value: renamed } });
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const retrieved = await client.schematics.symbols.retrieve({
          key: symbol.key,
        });
        expect(retrieved.name).toBe(renamed);
      });
    });
  });
});
