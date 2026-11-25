// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type group, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Theming } from "@synnaxlabs/lyra/theming";
import { theming } from "@synnaxlabs/pluto/ether";
import { color, uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, onTestFinished, vi } from "vitest";

const mocks = vi.hoisted(() => ({ symbolNodes: true }));

vi.mock("@/flags", async (importOriginal) => {
  const original = await importOriginal<{ FLAGS: typeof FLAGS }>();
  return {
    FLAGS: {
      ...original.FLAGS,
      get symbolNodes() {
        return mocks.symbolNodes;
      },
    },
  };
});

import { Schematic } from "@/feature/schematic";
import {
  client,
  createSymbolPayload,
  getEditSectionHeaderButton,
  getOverlayHandles,
  getPartListItem,
  getPreviewTransformWrapper,
  SYMBOL_FILE_DROP_PROMPT,
  symbolNames,
} from "@/feature/schematic/testutil";
import type { FLAGS } from "@/flags";
import {
  findButton,
  findLastButton,
  pressSaveTrigger,
  renderModalOpener,
} from "@/platform/modals/testutil";
import {
  fakePickedFile,
  getIconButton,
  interceptFilePicker,
  queryIcon,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
  mocks.symbolNodes = true;
});

// A shapeless SVG exercises the production load path.
const SHAPELESS_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><g id="grp"></g></svg>';

// The triggers provider identifies a key by KeyboardEvent.code, not by the printed
// character, so the zoom keys are pressed as Equal, Minus, and Digit0.
const pressKey = (code: string): void => {
  fireEvent.keyDown(window, { code });
  fireEvent.keyUp(window, { code });
};

const createParentGroup = async (): Promise<group.Group> =>
  await client.groups.create({
    scope: "schematic_symbol",
    name: uniqueName("edit_grp"),
  });

const openCreateModal = async (createKey?: string) => {
  const grp = await createParentGroup();
  const picker = interceptFilePicker();
  const handle = await renderModalOpener(
    Schematic.Symbol.Edit.useModal,
    [{ group: grp.key, createKey }],
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
  await screen.findByText("Parts");
};

const findNameInput = (): HTMLInputElement =>
  screen.getByPlaceholderText<HTMLInputElement>("Symbol name");

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
      await waitFor(async () => expect(await symbolNames(grp.key)).toContain(name));
      await waitFor(() =>
        expect(screen.queryByText(SYMBOL_FILE_DROP_PROMPT)).toBeNull(),
      );
    });

    it("creates the symbol on the shortcut its footer advertises", async () => {
      const { grp, picker } = await openCreateModal();
      await loadSVG(picker);
      const name = uniqueName("triggered_symbol");
      fireEvent.change(findNameInput(), { target: { value: name } });
      pressSaveTrigger();
      await waitFor(async () => expect(await symbolNames(grp.key)).toContain(name));
    });

    it("persists the symbol under a forced key when createKey is provided", async () => {
      const createKey = uuid.create();
      const { picker } = await openCreateModal(createKey);
      await loadSVG(picker);
      const name = uniqueName("healed_symbol");
      fireEvent.change(findNameInput(), { target: { value: name } });
      fireEvent.click(findButton("Create"));
      await waitFor(async () => {
        const created = await client.schematics.symbols.retrieve(createKey);
        expect(created.name).toBe(name);
      });
    });

    it("adds a part with no shapes and selects it", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Parts"));
      expect(await screen.findByRole("heading", { name: "Part 1" })).toBeDefined();
      expect(screen.getByText("0 shapes")).toBeDefined();
      expect(
        screen.getByText("Click shapes in the drawing to add or remove them."),
      ).toBeDefined();
    });

    it("saves a part stroke picked in the inspector", async () => {
      const createKey = uuid.create();
      const { picker } = await openCreateModal(createKey);
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Parts"));
      await screen.findByRole("heading", { name: "Part 1" });
      fireEvent.click(screen.getByLabelText("Stroke"));
      fireEvent.click(await screen.findByText("Color"));
      fireEvent.change(await screen.findByLabelText("Hex"), {
        target: { value: "ff0000" },
      });
      fireEvent.change(findNameInput(), {
        target: { value: uniqueName("colored_symbol") },
      });
      fireEvent.click(findButton("Create"));
      await waitFor(async () => {
        const created = await client.schematics.symbols.retrieve(createKey);
        const part = created.data.parts[0].key;
        const look = created.data.looks.find((l) => l.part === part);
        expect(look?.when).toBeUndefined();
        expect(look?.stroke).toEqual({
          type: "color",
          value: { type: "fixed", value: color.construct("#ff0000") },
        });
      });
    });

    it("saves the node color as the fill of the whole drawing", async () => {
      const createKey = uuid.create();
      const { picker } = await openCreateModal(createKey);
      await loadSVG(picker);
      expect(await screen.findByText("Looks here reach every shape.")).toBeDefined();
      fireEvent.click(screen.getByLabelText("Fill"));
      expect(findButton("State color").getAttribute("aria-disabled")).toBe("true");
      expect(findButton("Contrast").getAttribute("aria-disabled")).toBe("true");
      fireEvent.click(screen.getByText("Node color"));
      fireEvent.change(findNameInput(), {
        target: { value: uniqueName("node_colored_symbol") },
      });
      fireEvent.click(findButton("Create"));
      await waitFor(async () => {
        const created = await client.schematics.symbols.retrieve(createKey);
        expect(created.data.looks).toEqual([
          { key: expect.any(String), fill: { type: "node" } },
        ]);
      });
    });

    it("saves a fill for one state of an actuator", async () => {
      const createKey = uuid.create();
      const { picker } = await openCreateModal(createKey);
      await loadSVG(picker);
      fireEvent.click(await screen.findByText("Actuator"));
      fireEvent.click(getEditSectionHeaderButton("Parts"));
      await screen.findByRole("heading", { name: "Part 1" });
      const toggle = screen.getByLabelText("Fill by state");
      expect(toggle.getAttribute("aria-pressed")).toBe("false");
      fireEvent.click(toggle);
      expect(toggle.getAttribute("aria-pressed")).toBe("true");
      fireEvent.click(await screen.findByLabelText("Fill while Active"));
      fireEvent.click(screen.getByText("Node color"));
      fireEvent.change(findNameInput(), {
        target: { value: uniqueName("by_state_symbol") },
      });
      fireEvent.click(findButton("Create"));
      await waitFor(async () => {
        const created = await client.schematics.symbols.retrieve(createKey);
        const part = created.data.parts[0].key;
        expect(created.data.looks.filter((l) => l.part === part)).toEqual([
          {
            key: expect.any(String),
            part,
            when: { type: "equals", state: "active", values: [1] },
            fill: { type: "node" },
          },
        ]);
      });
    });

    it("adds a placeable handle at the symbol center", async () => {
      const { picker, handle } = await openCreateModal();
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Handles"));
      expect(await screen.findByText("Handle 1")).toBeDefined();
      expect(screen.getByText("(50%, 50%)")).toBeDefined();
      const [overlay] = getOverlayHandles(handle.baseElement, "left");
      expect(overlay.style.left).toBe("50%");
      expect(overlay.style.top).toBe("50%");
    });

    it("zooms the preview with keyboard shortcuts", async () => {
      const { picker } = await openCreateModal();
      await loadSVG(picker);
      await screen.findByText("100%");
      // Bare keys, since a browser reserves the Control chords for its own zoom.
      pressKey("Equal");
      expect(await screen.findByText("120%")).toBeDefined();
      pressKey("Digit0");
      expect(await screen.findByText("100%")).toBeDefined();
      pressKey("Minus");
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
      fireEvent.click(await screen.findByText("Actuator"));
      expect(await screen.findByText("Base")).toBeDefined();
      expect(screen.getByText("Active")).toBeDefined();
    });

    it("rejects a non-SVG file without opening the editor", async () => {
      const { picker } = await openCreateModal();
      fireEvent.click(await screen.findByText(SYMBOL_FILE_DROP_PROMPT));
      await waitFor(() => expect(picker.lastInput()).toBeDefined());
      picker.selectFiles([fakePickedFile("not_an_svg.txt", "hello")]);
      await waitFor(() => expect(screen.queryByText("Parts")).toBeNull());
      expect(screen.getByText(SYMBOL_FILE_DROP_PROMPT)).toBeDefined();
    });
  });

  describe("edit mode", () => {
    // openEditModal opens the editor on a new symbol with no state, as a client or the
    // v5 migration creates one.
    // The client that saves keeps its edits in its cache even when the Core refuses
    // them, so a new client reads what the Core stored.
    const retrieveStored = async (
      key: schematic.symbol.Key,
    ): Promise<schematic.symbol.Symbol> =>
      await createTestClient().schematics.symbols.retrieve(key);

    const openEditModal = async (markup?: string) => {
      const grp = await createParentGroup();
      const payload = createSymbolPayload(uniqueName("stateless"));
      if (markup != null) payload.data.drawing = { type: "svg", markup };
      const symbol = await client.schematics.symbols.create({
        ...payload,
        group: grp.key,
      });
      expect(symbol.data.states).toEqual([]);
      const handle = await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      await waitFor(() => expect(findNameInput().value).toBe(symbol.name), {
        timeout: 5000,
      });
      fireEvent.click(getEditSectionHeaderButton("Parts"));
      await screen.findByRole("heading", { name: "Part 1" });
      return { symbol, handle };
    };

    const retrieveStoredCommands = async (key: schematic.Key) =>
      (await createTestClient().schematics.retrieve(key)).nodes.map(({ config }) =>
        config.variant === "symbol" ? config.commands : undefined,
      );

    // openUsedSymbol opens the editor on a symbol that two schematics use. The nodes of
    // used bind the Vent command, and the node of unbound binds nothing.
    const openUsedSymbol = async () => {
      const grp = await createParentGroup();
      const payload = createSymbolPayload(uniqueName("vessel"));
      payload.data.commands = [
        { key: "vent", name: "Vent", value: { type: "fixed", value: 1 } },
      ];
      const symbol = await client.schematics.symbols.create({
        ...payload,
        group: grp.key,
      });
      const ch = await client.channels.create({
        name: uniqueName("vent_cmd"),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const project = await client.projects.create({ name: uniqueName("project") });
      const createNode = (key: string, label: string, bound: boolean) => ({
        key,
        position: { x: 0, y: 0 },
        zIndex: 0,
        config: schematic.nodeConfigZ.parse({
          variant: "symbol",
          spec: { type: "custom", key: symbol.key },
          label: { label },
          commands: bound ? [{ key: "vent", channel: ch.key }] : [],
        }),
      });
      const used = await client.schematics.create({
        project: project.key,
        name: uniqueName("plant"),
        nodes: [
          createNode("vessel", "Vessel 1", true),
          createNode("unlabeled", "", true),
        ],
      });
      const unbound = await client.schematics.create({
        project: project.key,
        name: uniqueName("spare"),
        nodes: [createNode("spare", "Spare vessel", false)],
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      await waitFor(() => expect(findNameInput().value).toBe(symbol.name), {
        timeout: 5000,
      });
      // Save waits for the uses of the symbol to load.
      await waitFor(() => expect(findButton("Save").disabled).toBe(false));
      return { symbol, used, unbound };
    };

    it("saves a new part and its stroke on a symbol with no state", async () => {
      const { symbol } = await openEditModal();
      fireEvent.click(screen.getByLabelText("Stroke"));
      fireEvent.click(await screen.findByText("Color"));
      fireEvent.change(await screen.findByLabelText("Hex"), {
        target: { value: "ff0000" },
      });
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const saved = await retrieveStored(symbol.key);
        expect(saved.data.states).toEqual([]);
        const part = saved.data.parts.find((p) => p.name === "Part 1");
        assertDefined(part);
        expect(saved.data.looks.find((l) => l.part === part.key)?.stroke).toEqual({
          type: "color",
          value: { type: "fixed", value: color.construct("#ff0000") },
        });
      });
    });

    it("bands a stroke in the theme of the preview", async () => {
      // The preview's theme leaf loads fonts on this thread, which has no self.fonts.
      vi.stubGlobal("fonts", document.fonts);
      onTestFinished(() => {
        vi.unstubAllGlobals();
      });
      const grp = await createParentGroup();
      const payload = createSymbolPayload(uniqueName("banded"));
      const symbol = await client.schematics.symbols.create({
        ...payload,
        group: grp.key,
        data: {
          ...payload.data,
          drawing: {
            type: "svg",
            markup:
              '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
              '<rect class="main" width="10" height="10"/></svg>',
          },
          parts: [{ key: "main", name: "Main", selectors: [".main"] }],
          looks: [
            {
              key: "stroke",
              part: "main",
              stroke: {
                type: "color",
                value: { type: "fixed", value: color.construct("#ffffff") },
              },
            },
          ],
        },
      });
      const handle = await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      const stroke = () =>
        getPreviewTransformWrapper(handle.baseElement).querySelector<SVGElement>(
          "rect.main",
        )?.style.stroke;
      // The light theme shows white as black.
      await waitFor(() => expect(stroke()).toBe("rgb(0, 0, 0)"), { timeout: 5000 });
      fireEvent.click(getIconButton(handle.baseElement, "light-mode"));
      await waitFor(() => expect(stroke()).toBe("rgb(255, 255, 255)"));
    });

    it("names an unnamed shape in the drawing when a click adds it to a part", async () => {
      const { symbol, handle } = await openEditModal(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
          '<rect data-region-id="region-1" width="10" height="10" fill="#ffffff"/>' +
          '<circle cx="5" cy="5" r="2" fill="#000000"/></svg>',
      );
      const circle = getPreviewTransformWrapper(handle.baseElement).querySelector(
        "circle",
      );
      assertDefined(circle);
      fireEvent.click(circle);
      await screen.findByText("1 shape");
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const saved = await retrieveStored(symbol.key);
        const part = saved.data.parts.find((p) => p.name === "Part 1");
        expect(part?.selectors).toEqual(['[data-region-id="region-2"]']);
        expect(saved.data.drawing.markup).toContain(
          '<circle cx="5" cy="5" r="2" fill="#000000" data-region-id="region-2"/>',
        );
      });
    });

    it("asks before a save removes bindings from a schematic that uses the symbol", async () => {
      const { used, unbound } = await openUsedSymbol();
      fireEvent.click(screen.getByText("Actuator"));
      fireEvent.click(findButton("Save"));
      await screen.findByText(used.name);
      await screen.findByText("Vessel 1 loses the Vent binding.");
      await screen.findByText("A node with no label loses the Vent binding.");
      await screen.findByText(unbound.name);
      await screen.findByText("Each node keeps its bindings.");
      fireEvent.click(findLastButton("Save"));
      await waitFor(async () =>
        expect(await retrieveStoredCommands(used.key)).toEqual([[], []]),
      );
    });

    it("keeps the symbol and its bindings when the save confirm is canceled", async () => {
      const { symbol, used } = await openUsedSymbol();
      fireEvent.click(screen.getByText("Actuator"));
      fireEvent.click(findButton("Save"));
      await screen.findByText("Vessel 1 loses the Vent binding.");
      fireEvent.click(findButton("Cancel"));
      await waitFor(() =>
        expect(screen.queryByText("Vessel 1 loses the Vent binding.")).toBeNull(),
      );
      expect(findNameInput().value).toBe(symbol.name);
      expect((await retrieveStored(symbol.key)).data.variant).toBe("static");
      const commands = await retrieveStoredCommands(used.key);
      expect(commands.map((c) => c?.length)).toEqual([1, 1]);
    });

    it("saves a zoom and a rename of a symbol in use without asking", async () => {
      const { symbol } = await openUsedSymbol();
      await screen.findByText("100%");
      pressKey("Equal");
      await screen.findByText("120%");
      const renamed = uniqueName("renamed_vessel");
      fireEvent.change(findNameInput(), { target: { value: renamed } });
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).name).toBe(renamed),
      );
      const stored = await retrieveStored(symbol.key);
      expect(stored.data.previewViewport?.zoom).toBeCloseTo(1.2);
    });

    it("loads the existing symbol and renames it on save", async () => {
      const grp = await createParentGroup();
      const original = uniqueName("existing");
      const symbol = await client.schematics.symbols.create({
        ...createSymbolPayload(original),
        group: grp.key,
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      // The modal suspends on the symbol retrieve, which can outlast the default
      // waitFor timeout on a loaded CI machine.
      const nameInput = await waitFor(
        () => {
          const input = findNameInput();
          expect(input.value).toBe(original);
          return input;
        },
        { timeout: 5000 },
      );
      const renamed = uniqueName("renamed_symbol");
      fireEvent.change(nameInput, { target: { value: renamed } });
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const retrieved = await client.schematics.symbols.retrieve(symbol.key);
        expect(retrieved.name).toBe(renamed);
      });
    });

    // openStored opens the editor on a new symbol, and edits nothing.
    const openStored = async () => {
      const grp = await createParentGroup();
      const symbol = await client.schematics.symbols.create({
        ...createSymbolPayload(uniqueName("stored")),
        group: grp.key,
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      await waitFor(() => expect(findNameInput().value).toBe(symbol.name), {
        timeout: 5000,
      });
      return symbol;
    };

    const closeEditor = (): void => {
      fireEvent.click(screen.getByRole("button", { name: "Close" }));
    };

    it("closes at once when nothing changed", async () => {
      await openStored();
      closeEditor();
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Symbol name")).toBeNull(),
      );
      expect(screen.queryByText("Discard unsaved edits?")).toBeNull();
    });

    it("closes at once when an edit is undone", async () => {
      const symbol = await openStored();
      fireEvent.change(findNameInput(), { target: { value: uniqueName("renamed") } });
      fireEvent.change(findNameInput(), { target: { value: symbol.name } });
      closeEditor();
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Symbol name")).toBeNull(),
      );
      expect(screen.queryByText("Discard unsaved edits?")).toBeNull();
    });

    it("closes at once on Escape after a zoom of the preview", async () => {
      await openStored();
      pressKey("Equal");
      // A modal takes an Escape only from an element inside it.
      const zoom = await screen.findByText("120%");
      fireEvent.keyDown(zoom, { code: "Escape", key: "Escape" });
      fireEvent.keyUp(zoom, { code: "Escape", key: "Escape" });
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Symbol name")).toBeNull(),
      );
      expect(screen.queryByText("Discard unsaved edits?")).toBeNull();
    });

    it("asks before it drops unsaved edits on close", async () => {
      const symbol = await openStored();
      const renamed = uniqueName("renamed");
      fireEvent.change(findNameInput(), { target: { value: renamed } });
      closeEditor();
      expect(await screen.findByText("Discard unsaved edits?")).toBeDefined();
      fireEvent.click(findButton("Keep editing"));
      await waitFor(() =>
        expect(screen.queryByText("Discard unsaved edits?")).toBeNull(),
      );
      expect(findNameInput().value).toBe(renamed);
      closeEditor();
      fireEvent.click(await waitFor(() => findButton("Discard")));
      await waitFor(() =>
        expect(screen.queryByPlaceholderText("Symbol name")).toBeNull(),
      );
      expect((await retrieveStored(symbol.key)).name).toBe(symbol.name);
    });

    const BODY: schematic.symbol.Part = {
      key: "body",
      name: "Body",
      selectors: ["#a"],
    };
    const GREEN = color.construct("#2fb36b");

    const createState = (enumeration?: string): schematic.symbol.State => ({
      type: "enumeration",
      key: "active",
      name: "Active",
      enumeration,
      match: { type: "exact" },
      default: 0,
    });

    // createEnumeration creates an enumeration whose one entry, Closed at 0, is green.
    const createEnumeration = async () =>
      await client.enumerations.create({
        name: uniqueName("valve"),
        entries: [
          {
            key: "closed",
            value: 0,
            name: "Closed",
            color: { type: "fixed", value: GREEN },
          },
        ],
      });

    // openPainted opens the editor on a new symbol whose one part, Body, selects #a,
    // and selects Body.
    const openPainted = async (data: Partial<schematic.symbol.Spec>) => {
      const grp = await createParentGroup();
      const payload = createSymbolPayload(uniqueName("painted"));
      const symbol = await client.schematics.symbols.create({
        ...payload,
        group: grp.key,
        data: { ...payload.data, parts: [BODY], ...data },
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      fireEvent.click(await screen.findByText("Body"));
      await screen.findByRole("heading", { name: "Body" });
      return symbol;
    };

    it("keeps Drawing from a part that the whole drawing paints", async () => {
      await openPainted({ looks: [{ key: "node", fill: { type: "node" } }] });
      fireEvent.click(screen.getByLabelText("Fill"));
      await waitFor(() =>
        expect(findButton("Drawing").getAttribute("aria-disabled")).toBe("true"),
      );
    });

    it("gives an overriding part the paint of the whole drawing back", async () => {
      const node: schematic.symbol.Look = { key: "node", fill: { type: "node" } };
      const symbol = await openPainted({
        states: [createState()],
        looks: [
          node,
          {
            key: "red",
            part: "body",
            when: { type: "equals", state: "active", values: [0] },
            fill: {
              type: "color",
              value: { type: "fixed", value: color.construct("#ff0000") },
            },
          },
        ],
      });
      fireEvent.click(screen.getByLabelText("Fill"));
      const dialog = await waitFor(() => {
        const el = document.querySelector<HTMLElement>(
          ".console-schematic-paint-dialog",
        );
        assertDefined(el);
        return el;
      });
      fireEvent.click(within(dialog).getByText("Whole drawing"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([node]),
      );
    });

    it("hides Node color on a static symbol while symbol nodes are off", async () => {
      mocks.symbolNodes = false;
      await openPainted({});
      fireEvent.click(screen.getByLabelText("Fill"));
      await screen.findByText("Contrast");
      expect(screen.queryByText("Node color")).toBeNull();
    });

    it("offers Node color on an actuator while symbol nodes are off", async () => {
      mocks.symbolNodes = false;
      await openPainted({ variant: "actuator", states: [createState()] });
      fireEvent.click(screen.getByLabelText("Fill"));
      expect(await screen.findByText("Node color")).toBeDefined();
    });

    it("draws a node color paint in a sample color in the preview", async () => {
      await openPainted({
        parts: [{ ...BODY, selectors: ["#body"] }],
        looks: [{ key: "node", part: "body", fill: { type: "node" } }],
      });
      const { primary } = Theming.themeZ.parse(Theming.SYNNAX_DARK).colors;
      await waitFor(() => {
        const shape = getPreviewTransformWrapper(document.body).querySelector("#body");
        expect(shape?.getAttribute("fill")).toBe(color.hex(primary.z));
      });
    });

    it("names the field and the kind in the tooltip of a paint", async () => {
      await openPainted({});
      fireEvent.pointerEnter(screen.getByLabelText("Fill"));
      expect((await screen.findByRole("tooltip")).textContent).toBe("Fill: Drawing");
    });

    it("saves the state color and shows the default entry where no entry matches", async () => {
      const enumeration = await createEnumeration();
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState(enumeration.key)],
      });
      fireEvent.click(await screen.findByText("Active"));
      const fill = screen.getByLabelText("Fill");
      fireEvent.click(fill);
      fireEvent.click(await screen.findByText("State color"));
      await waitFor(() =>
        expect(fill.style.getPropertyValue("--pluto-bg")).toBe(color.cssString(GREEN)),
      );
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const saved = await retrieveStored(symbol.key);
        expect(saved.data.looks).toEqual([
          {
            key: expect.any(String),
            part: "body",
            fill: { type: "entry", state: "active" },
          },
        ]);
      });
    });

    it("saves a contrast paint on the state", async () => {
      const enumeration = await createEnumeration();
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState(enumeration.key)],
      });
      fireEvent.click(screen.getByLabelText("Fill"));
      fireEvent.click(await screen.findByText("Contrast"));
      await waitFor(() =>
        expect(queryIcon(screen.getByLabelText("Fill"), "contrast")).not.toBeNull(),
      );
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([
          {
            key: expect.any(String),
            part: "body",
            fill: { type: "contrast", state: "active" },
          },
        ]),
      );
    });

    it("shows the paint that the whole drawing gives a part with none", async () => {
      await openPainted({ looks: [{ key: "node", fill: { type: "node" } }] });
      expect(queryIcon(screen.getByLabelText("Fill"), "node")).not.toBeNull();
    });

    const byState = (label: string) =>
      fireEvent.click(screen.getByLabelText(`${label} by state`));
    const atBase: schematic.symbol.Condition = {
      type: "equals",
      state: "active",
      values: [0],
    };

    it("edits the paint at Base of a static symbol with one control", async () => {
      const symbol = await openPainted({
        states: [createState()],
        looks: [
          {
            key: "red",
            part: "body",
            when: { type: "equals", state: "active", values: [0] },
            fill: {
              type: "color",
              value: { type: "fixed", value: color.construct("#ff0000") },
            },
          },
        ],
      });
      expect(screen.queryByText("By state")).toBeNull();
      fireEvent.click(screen.getByLabelText("Fill"));
      fireEvent.click(await screen.findByText("Drawing"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const saved = await retrieveStored(symbol.key);
        expect(saved.data.looks).toEqual([]);
      });
    });

    it("keeps By state when the last paint of a part at a value is cleared", async () => {
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState()],
        looks: [
          {
            key: "red",
            part: "body",
            when: atBase,
            fill: {
              type: "color",
              value: { type: "fixed", value: color.construct("#ff0000") },
            },
          },
        ],
      });
      fireEvent.click(screen.getByLabelText("Fill while Base"));
      fireEvent.click(await screen.findByText("Drawing"));
      expect(screen.getByLabelText("Fill while Active")).toBeDefined();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([]),
      );
    });

    it("leaves By state when the symbol turns static", async () => {
      await openPainted({ variant: "actuator", states: [createState()] });
      byState("Fill");
      expect(screen.getByLabelText("Fill while Base")).toBeDefined();
      fireEvent.click(screen.getByText("Static"));
      await waitFor(() =>
        expect(screen.queryByLabelText("Fill while Base")).toBeNull(),
      );
      expect(screen.getByLabelText("Fill")).toBeDefined();
    });

    it("stays out of By state when the symbol turns static and back", async () => {
      await openPainted({ variant: "actuator", states: [createState()] });
      byState("Fill");
      fireEvent.click(screen.getByText("Static"));
      await waitFor(() =>
        expect(screen.queryByLabelText("Fill while Base")).toBeNull(),
      );
      fireEvent.click(screen.getByText("Actuator"));
      await screen.findAllByText("Active");
      expect(screen.queryByLabelText("Fill while Base")).toBeNull();
      expect(screen.getByLabelText("Fill")).toBeDefined();
    });

    it("shows the paint of the whole drawing at the preview value", async () => {
      await openPainted({
        variant: "actuator",
        states: [createState()],
        looks: [{ key: "node", when: atBase, fill: { type: "node" } }],
      });
      expect(queryIcon(screen.getByLabelText("Fill"), "node")).not.toBeNull();
    });

    it("starts a picked color from the color of the drawing", async () => {
      const symbol = await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
            '<rect id="a" width="10" height="10" fill="#2fb36b"/></svg>',
        },
      });
      fireEvent.click(screen.getByLabelText("Fill"));
      fireEvent.click(await screen.findByText("Color"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const saved = await retrieveStored(symbol.key);
        expect(saved.data.looks.find((l) => l.part === "body")?.fill).toEqual({
          type: "color",
          value: { type: "fixed", value: GREEN },
        });
      });
    });

    it("selects the whole drawing again when the selected part is removed", async () => {
      const { symbol, handle } = await openEditModal(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
          '<rect id="a" width="10" height="10"/></svg>',
      );
      fireEvent.click(getIconButton(getPartListItem("Part 1"), "close"));
      await waitFor(() => expect(screen.queryByText("Part 1")).toBeNull());
      expect(
        getPartListItem("Whole drawing").classList.contains("lyra--selected"),
      ).toBe(true);
      const shape = getPreviewTransformWrapper(handle.baseElement).querySelector("#a");
      assertDefined(shape);
      fireEvent.click(shape);
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([]),
      );
    });

    it("counts the looks of a part beside its shapes", async () => {
      await openPainted({ looks: [{ key: "l", part: "body", hidden: true }] });
      expect(await screen.findByText("1 shape · 1 look")).toBeDefined();
    });

    it("counts the shapes of each part in the singular and the plural", async () => {
      const grp = await createParentGroup();
      const payload = createSymbolPayload(uniqueName("counted"));
      const parts = [
        { key: "one", name: "One", selectors: ["#a"] },
        { key: "two", name: "Two", selectors: ["#a", "#b"] },
      ];
      const symbol = await client.schematics.symbols.create({
        ...payload,
        group: grp.key,
        data: { ...payload.data, parts },
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      expect(await screen.findByText("1 shape")).toBeDefined();
      expect(screen.getByText("2 shapes")).toBeDefined();
    });
  });
});
