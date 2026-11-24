// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { group, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Theming } from "@synnaxlabs/lyra/theming";
import { theming } from "@synnaxlabs/pluto/ether";
import { color, uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, onTestFinished, vi } from "vitest";
import { page, server, userEvent } from "vitest/browser";

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
  getInspector,
  getOverlayHandles,
  getPartListItem,
  getPreviewTransformWrapper,
  SYMBOL_FILE_DROP_PROMPT,
  symbolNames,
} from "@/feature/schematic/testutil";
import type { FLAGS } from "@/flags";
import {
  findButton,
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
      expect(
        await screen.findByText(
          "Looks here reach every shape. Click a shape to select its part or add one.",
        ),
      ).toBeDefined();
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

    // createWithPart opens the editor on a new drawing with one new part, Part 1.
    const createWithPart = async () => {
      const createKey = uuid.create();
      const { picker } = await openCreateModal(createKey);
      await loadSVG(picker);
      fireEvent.click(getEditSectionHeaderButton("Parts"));
      await screen.findByRole("heading", { name: "Part 1" });
      fireEvent.change(findNameInput(), {
        target: { value: uniqueName("moving_symbol") },
      });
      return createKey;
    };

    const savedLooks = async (createKey: string) => {
      const created = await client.schematics.symbols.retrieve(createKey);
      const part = created.data.parts[0].key;
      return created.data.looks.filter((l) => l.part === part);
    };

    it("saves a hidden part that blinks", async () => {
      const createKey = await createWithPart();
      fireEvent.click(screen.getByLabelText("Visible"));
      fireEvent.click(screen.getByLabelText("Blink"));
      fireEvent.click(findButton("Create"));
      await waitFor(async () =>
        expect(await savedLooks(createKey)).toEqual([
          expect.objectContaining({ hidden: true, blink: { period: 1e9 } }),
        ]),
      );
    });

    it("offers the loop motions in a Loop group", async () => {
      await createWithPart();
      expect(screen.getByText("Loop")).toBeDefined();
      expect(screen.getByText("Spin")).toBeDefined();
      expect(screen.getByText("Flow")).toBeDefined();
      expect(screen.queryByText("Follow a value")).toBeNull();
    });

    it("saves a spin at the rate that its field shows whole", async () => {
      const createKey = await createWithPart();
      fireEvent.click(screen.getByText("Spin"));
      const rate = await screen.findByLabelText<HTMLInputElement>("Motion rate");
      expect(rate.value).toBe("0.5");
      expect(rate.classList.contains("lyra--cut")).toBe(false);
      fireEvent.change(rate, { target: { value: "2" } });
      fireEvent.blur(rate);
      fireEvent.click(findButton("Create"));
      await waitFor(async () =>
        expect(await savedLooks(createKey)).toEqual([
          expect.objectContaining({ motion: { type: "spin", rate: 2 } }),
        ]),
      );
    });

    it("keeps the rate when a spin turns into a flow", async () => {
      const createKey = await createWithPart();
      fireEvent.click(screen.getByText("Spin"));
      const rate = await screen.findByLabelText<HTMLInputElement>("Motion rate");
      fireEvent.change(rate, { target: { value: "2" } });
      fireEvent.blur(rate);
      fireEvent.click(screen.getByText("Flow"));
      fireEvent.click(findButton("Create"));
      await waitFor(async () =>
        expect(await savedLooks(createKey)).toEqual([
          expect.objectContaining({ motion: { type: "flow", rate: 2 } }),
        ]),
      );
    });

    it("saves a spin at a negative rate", async () => {
      const createKey = await createWithPart();
      fireEvent.click(screen.getByText("Spin"));
      const rate = await screen.findByLabelText<HTMLInputElement>("Motion rate");
      fireEvent.change(rate, { target: { value: "-1" } });
      fireEvent.blur(rate);
      fireEvent.click(findButton("Create"));
      await waitFor(async () =>
        expect(await savedLooks(createKey)).toEqual([
          expect.objectContaining({ motion: { type: "spin", rate: -1 } }),
        ]),
      );
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

    // openDrawing opens the editor on a new symbol that draws markup and has no parts.
    const openDrawing = async (markup: string) => {
      const grp = await createParentGroup();
      const payload = createSymbolPayload(uniqueName("drawn"));
      const symbol = await client.schematics.symbols.create({
        ...payload,
        group: grp.key,
        data: { ...payload.data, drawing: { type: "svg", markup } },
      });
      await renderModalOpener(
        Schematic.Symbol.Edit.useModal,
        [{ symbolKey: symbol.key, group: grp.key }],
        { client, additionalRegistry: theming.REGISTRY },
      );
      await screen.findByRole("heading", { name: "Whole drawing" });
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

    it("shows a part that the whole drawing hides", async () => {
      const symbol = await openPainted({ looks: [{ key: "hide", hidden: true }] });
      const visible = screen.getByLabelText<HTMLInputElement>("Visible");
      expect(visible.checked).toBe(false);
      fireEvent.click(visible);
      await waitFor(() => expect(visible.checked).toBe(true));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([
          { key: "hide", hidden: true },
          { key: expect.any(String), part: "body", hidden: false },
        ]),
      );
    });

    it("keeps the blink and motion that the whole drawing gives a part", async () => {
      await openPainted({
        looks: [
          { key: "all", blink: { period: 1e9 }, motion: { type: "spin", rate: 1 } },
        ],
      });
      const blink = screen.getByLabelText<HTMLInputElement>("Blink");
      expect(blink.checked).toBe(true);
      expect(blink.disabled).toBe(true);
      for (const kind of ["None", "Spin", "Flow"])
        expect(findButton(kind).getAttribute("aria-disabled")).toBe("true");
      expect(screen.getByLabelText<HTMLInputElement>("Motion rate").disabled).toBe(
        true,
      );
    });

    it("clears the own blink and motion of a part under those of the drawing", async () => {
      const all: schematic.symbol.Look = {
        key: "all",
        blink: { period: 1e9 },
        motion: { type: "spin", rate: 1 },
      };
      const symbol = await openPainted({
        looks: [
          all,
          {
            key: "own",
            part: "body",
            blink: { period: 1e9 },
            motion: { type: "flow", rate: 1 },
          },
        ],
      });
      const blink = () => screen.getByLabelText<HTMLInputElement>("Blink");
      expect(blink().disabled).toBe(false);
      fireEvent.click(blink());
      await waitFor(() => expect(blink().disabled).toBe(true));
      fireEvent.click(findButton("None"));
      await waitFor(() =>
        expect(findButton("None").getAttribute("aria-disabled")).toBe("true"),
      );
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([all]),
      );
    });

    const byState = (label: string) =>
      fireEvent.click(screen.getByLabelText(`${label} by state`));
    const atActive: schematic.symbol.Condition = {
      type: "equals",
      state: "active",
      values: [1],
    };
    const atBase: schematic.symbol.Condition = {
      type: "equals",
      state: "active",
      values: [0],
    };

    it("keeps a part shown at every value where the drawing hides it", async () => {
      const hide: schematic.symbol.Look = { key: "hide", when: atActive, hidden: true };
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState()],
        looks: [hide],
      });
      const visible = () => screen.getByLabelText<HTMLInputElement>("Visible");
      fireEvent.click(visible());
      await waitFor(() => expect(visible().checked).toBe(false));
      fireEvent.click(visible());
      await waitFor(() => expect(visible().checked).toBe(true));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([
          hide,
          { key: expect.any(String), part: "body", hidden: false },
        ]),
      );
    });

    it("shows and hides a part at each value of a state", async () => {
      const hide: schematic.symbol.Look = { key: "hide", when: atActive, hidden: true };
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState()],
        looks: [hide],
      });
      byState("Visible");
      const visible = (v: string) =>
        screen.getByLabelText<HTMLInputElement>(`Visible while ${v}`);
      expect(visible("Active").checked).toBe(false);
      fireEvent.click(visible("Active"));
      await waitFor(() => expect(visible("Active").checked).toBe(true));
      fireEvent.click(visible("Base"));
      await waitFor(() => expect(visible("Base").checked).toBe(false));
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const { looks } = (await retrieveStored(symbol.key)).data;
        expect(looks).toHaveLength(3);
        expect(looks).toEqual(
          expect.arrayContaining([
            hide,
            { key: expect.any(String), part: "body", when: atActive, hidden: false },
            { key: expect.any(String), part: "body", when: atBase, hidden: true },
          ]),
        );
      });
    });

    it("clears the hide of a part at a value and keeps the values shown", async () => {
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState()],
        looks: [{ key: "hide", part: "body", when: atBase, hidden: true }],
      });
      const visible = () =>
        screen.getByLabelText<HTMLInputElement>("Visible while Base");
      expect(visible().checked).toBe(false);
      fireEvent.click(visible());
      await waitFor(() => expect(visible().checked).toBe(true));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([]),
      );
    });

    it("sets the blink and motion of a part at one value of a state", async () => {
      const all: schematic.symbol.Look = {
        key: "all",
        when: atActive,
        blink: { period: 1e9 },
      };
      const symbol = await openPainted({
        variant: "actuator",
        states: [createState()],
        looks: [all],
      });
      byState("Blink");
      const blink = (v: string) =>
        screen.getByLabelText<HTMLInputElement>(`Blink while ${v}`);
      expect(blink("Active").checked).toBe(true);
      expect(blink("Active").disabled).toBe(true);
      fireEvent.click(blink("Base"));
      await waitFor(() => expect(blink("Base").checked).toBe(true));
      byState("Motion");
      fireEvent.click(
        within(screen.getByLabelText("Motion while Base")).getByText("Spin"),
      );
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([
          all,
          {
            key: expect.any(String),
            part: "body",
            when: atBase,
            blink: expect.any(Object),
            motion: expect.objectContaining({ type: "spin" }),
          },
        ]),
      );
    });

    it("fits the motion picker and its By state button in the inspector", async () => {
      await openPainted({ variant: "actuator", states: [createState()] });
      const right = getInspector().getBoundingClientRect().right;
      for (const el of [screen.getByLabelText("Motion by state"), findButton("Flow")])
        expect(el.getBoundingClientRect().right).toBeLessThanOrEqual(right);
    });

    it("shows a motion that follows a value and replaces it with a loop", async () => {
      const level: schematic.symbol.Look = {
        key: "level",
        part: "body",
        motion: { type: "level", state: "tank", side: "bottom" },
      };
      const symbol = await openPainted({
        states: [
          { type: "number", key: "tank", name: "Tank", bounds: { lower: 0, upper: 1 } },
        ],
        looks: [level],
      });
      expect(findButton("Fill").classList.contains("lyra--selected")).toBe(true);
      expect(findButton("Fill").getAttribute("aria-disabled")).toBe("true");
      fireEvent.click(findButton("Spin"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([
          { ...level, motion: expect.objectContaining({ type: "spin" }) },
        ]),
      );
    });

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

    it("selects the part when a save meets a level motion of two shapes", async () => {
      await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
            '<rect id="a" width="10" height="10"/>' +
            '<rect id="b" x="10" width="10" height="10"/></svg>',
        },
        parts: [{ ...BODY, selectors: ["#a", "#b"] }],
        states: [
          { type: "number", key: "tank", name: "Tank", bounds: { lower: 0, upper: 1 } },
        ],
        looks: [
          {
            key: "level",
            part: "body",
            motion: { type: "level", state: "tank", side: "bottom" },
          },
        ],
      });
      fireEvent.click(screen.getByText("Whole drawing"));
      await screen.findByRole("heading", { name: "Whole drawing" });
      expect(findButton("Save").getAttribute("aria-disabled")).toBeNull();
      fireEvent.click(findButton("Save"));
      await screen.findByRole("heading", { name: "Body" });
      expect(screen.getByText("Fill motion needs one shape")).toBeDefined();
      fireEvent.click(screen.getByText("Spin"));
      await waitFor(() =>
        expect(screen.queryByText("Fill motion needs one shape")).toBeNull(),
      );
    });

    it("saves once the level motion that replaces a clip-path goes", async () => {
      const symbol = await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
            '<rect id="a" width="10" height="10" clip-path="inset(1px)"/></svg>',
        },
        states: [
          { type: "number", key: "tank", name: "Tank", bounds: { lower: 0, upper: 1 } },
        ],
        looks: [
          {
            key: "level",
            part: "body",
            motion: { type: "level", state: "tank", side: "bottom" },
          },
        ],
      });
      const reason = "Fill motion replaces a clip-path";
      expect(screen.getByText(reason)).toBeDefined();
      fireEvent.click(screen.getByText("None"));
      await waitFor(() => expect(screen.queryByText(reason)).toBeNull());
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.looks).toEqual([]),
      );
    });

    const TWO_RECTS =
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
      '<rect id="a" width="10" height="10"/><rect id="b" x="10" width="10" height="10"/>' +
      "</svg>";

    const GROUPED =
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
      '<g id="g"><rect id="c" width="10" height="10"/>' +
      '<rect id="d" x="10" width="10" height="10"/></g></svg>';

    const findShape = (id: string): Element => {
      const shape = getPreviewTransformWrapper(document.body).querySelector(`#${id}`);
      assertDefined(shape);
      return shape;
    };

    it("marks the shapes of the selected part in the preview", async () => {
      await openPainted({ drawing: { type: "svg", markup: TWO_RECTS } });
      const selected = "console-schematic__svg-region--selected";
      await waitFor(() =>
        expect(findShape("a").classList.contains(selected)).toBe(true),
      );
      expect(findShape("b").classList.contains(selected)).toBe(false);
      fireEvent.click(screen.getByText("Whole drawing"));
      await waitFor(() =>
        expect(findShape("a").classList.contains(selected)).toBe(false),
      );
    });

    it("makes a part of a shape clicked with the whole drawing selected", async () => {
      const symbol = await openDrawing(TWO_RECTS);
      fireEvent.click(findShape("b"));
      expect(await screen.findByRole("heading", { name: "Part 1" })).toBeDefined();
      fireEvent.click(findButton("Save"));
      await waitFor(async () => {
        const saved = await retrieveStored(symbol.key);
        expect(saved.data.parts).toEqual([
          { key: expect.any(String), name: "Part 1", selectors: ["#b"] },
        ]);
      });
    });

    it("keeps the parts when a Shift+drag pan ends on a shape", async () => {
      const symbol = await openPainted({ drawing: { type: "svg", markup: TWO_RECTS } });
      fireEvent.click(findShape("b"), { shiftKey: true });
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([BODY]),
      );
    });

    it("selects the part of a shape clicked with the whole drawing selected", async () => {
      const symbol = await openPainted({ drawing: { type: "svg", markup: TWO_RECTS } });
      fireEvent.click(screen.getByText("Whole drawing"));
      await screen.findByRole("heading", { name: "Whole drawing" });
      fireEvent.click(findShape("a"));
      expect(await screen.findByRole("heading", { name: "Body" })).toBeDefined();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([BODY]),
      );
    });

    it("keeps the filter of the drawing on a shape at rest", async () => {
      await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
            '<defs><filter id="f"><feGaussianBlur stdDeviation="1"/></filter></defs>' +
            '<rect id="a" width="10" height="10"/>' +
            '<rect id="b" x="10" width="10" height="10" filter="url(#f)"/></svg>',
        },
      });
      await waitFor(() =>
        expect(getComputedStyle(findShape("b")).filter).toContain("url("),
      );
    });

    it("keeps the filter that a style sets on a shape at rest", async () => {
      await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
            '<defs><filter id="f"><feGaussianBlur stdDeviation="1"/></filter></defs>' +
            '<rect id="a" width="10" height="10"/>' +
            '<rect id="b" x="10" width="10" height="10" style="filter:url(#f)"/></svg>',
        },
      });
      await waitFor(() =>
        expect(getComputedStyle(findShape("b")).filter).toContain("url("),
      );
    });

    it("keeps the filter of the drawing on a selected shape", async () => {
      await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
            '<defs><filter id="f"><feGaussianBlur stdDeviation="1"/></filter></defs>' +
            '<rect id="b" width="10" height="10" filter="url(#f)"/></svg>',
        },
        parts: [{ ...BODY, selectors: ["#b"] }],
      });
      await waitFor(() => {
        const { filter } = getComputedStyle(findShape("b"));
        expect(filter).toContain("url(");
        expect(filter).toContain("drop-shadow");
      });
    });

    it("marks a selected shape whose style sets a filter", async () => {
      await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
            '<defs><filter id="f"><feGaussianBlur stdDeviation="1"/></filter></defs>' +
            '<rect id="b" width="10" height="10" style="filter:url(#f)"/></svg>',
        },
        parts: [{ ...BODY, selectors: ["#b"] }],
      });
      await waitFor(() => {
        const { filter } = getComputedStyle(findShape("b"));
        expect(filter).toContain("url(");
        expect(filter).toContain("drop-shadow");
      });
    });

    it("marks a selected group and not each shape inside it", async () => {
      await openPainted({
        drawing: { type: "svg", markup: GROUPED },
        parts: [{ ...BODY, selectors: ["#g"] }],
      });
      await waitFor(() =>
        expect(getComputedStyle(findShape("g")).filter).toContain("drop-shadow"),
      );
      expect(getComputedStyle(findShape("c")).filter).not.toContain("drop-shadow");
    });

    it("keeps the mark of a filled selected shape under the pointer", async () => {
      const { width, height } = server.config.browser.viewport;
      onTestFinished(async () => await page.viewport(width, height));
      // The pointer reaches the shape only when the editor fits the viewport.
      await page.viewport(1300, 900);
      await openPainted({
        drawing: {
          type: "svg",
          markup: GROUPED.replace('id="c"', 'id="c" fill="red"'),
        },
        parts: [{ ...BODY, selectors: ["#c"] }],
      });
      const shape = findShape("c");
      await userEvent.hover(shape);
      await waitFor(() => {
        const { filter } = getComputedStyle(shape);
        expect(filter).toContain("drop-shadow");
        expect(filter).toContain("brightness(1.3)");
      });
    });

    it("selects the part that holds the group of a shape clicked", async () => {
      await openPainted({
        drawing: { type: "svg", markup: GROUPED },
        parts: [{ ...BODY, selectors: ["#g"] }],
      });
      fireEvent.click(screen.getByText("Whole drawing"));
      await screen.findByRole("heading", { name: "Whole drawing" });
      fireEvent.click(findShape("c"));
      expect(await screen.findByRole("heading", { name: "Body" })).toBeDefined();
      expect(screen.queryByText("Part 1")).toBeNull();
    });

    it("selects the part that holds the group of a clicked shape from another part", async () => {
      await openPainted({
        drawing: {
          type: "svg",
          markup: GROUPED.replace("</svg>", '<rect id="e" y="10" width="10"/></svg>'),
        },
        parts: [
          { ...BODY, selectors: ["#e"] },
          { key: "pump", name: "Pump", selectors: ["#g"] },
        ],
      });
      fireEvent.click(findShape("c"));
      expect(await screen.findByRole("heading", { name: "Pump" })).toBeDefined();
      expect(getPartListItem("Body").textContent).toContain("1 shape");
    });

    it("keeps the part that a double click selects", async () => {
      const symbol = await openPainted({
        drawing: { type: "svg", markup: GROUPED },
        parts: [{ ...BODY, selectors: ["#c"] }],
      });
      fireEvent.click(screen.getByText("Whole drawing"));
      await screen.findByRole("heading", { name: "Whole drawing" });
      fireEvent.click(findShape("c"), { detail: 1 });
      await screen.findByRole("heading", { name: "Body" });
      fireEvent.click(findShape("c"), { detail: 2 });
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([
          { ...BODY, selectors: ["#c"] },
        ]),
      );
    });

    it("takes each click of a double click while a part is selected", async () => {
      const symbol = await openPainted({ drawing: { type: "svg", markup: TWO_RECTS } });
      fireEvent.click(findShape("b"), { detail: 1 });
      fireEvent.click(findShape("a"), { detail: 2 });
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([
          { ...BODY, selectors: ["#b"] },
        ]),
      );
    });

    it("takes the group out of the part when a click lands inside it", async () => {
      const symbol = await openPainted({
        drawing: { type: "svg", markup: GROUPED },
        parts: [{ ...BODY, selectors: ["#g"] }],
      });
      fireEvent.click(findShape("c"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([
          { ...BODY, selectors: [] },
        ]),
      );
    });

    it("takes a text out of a part that holds only its span", async () => {
      const symbol = await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
            '<text id="t" y="8"><tspan id="s">Tank</tspan></text></svg>',
        },
        parts: [{ ...BODY, selectors: ["#s"] }],
      });
      fireEvent.click(findShape("s"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([
          { ...BODY, selectors: [] },
        ]),
      );
    });

    it("toggles the text of a span clicked", async () => {
      const symbol = await openPainted({
        drawing: {
          type: "svg",
          markup:
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 10">' +
            '<text id="t" y="8"><tspan id="s">Tank</tspan></text></svg>',
        },
        parts: [{ ...BODY, selectors: ["#t"] }],
      });
      fireEvent.click(findShape("s"));
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([
          { ...BODY, selectors: [] },
        ]),
      );
    });

    it("adds only the clicked shape of a group to a part", async () => {
      const symbol = await openDrawing(GROUPED);
      fireEvent.click(findShape("c"));
      await screen.findByRole("heading", { name: "Part 1" });
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect((await retrieveStored(symbol.key)).data.parts).toEqual([
          { key: expect.any(String), name: "Part 1", selectors: ["#c"] },
        ]),
      );
    });

    it("selects the whole drawing again when the selected part is removed", async () => {
      await openEditModal(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">' +
          '<rect id="a" width="10" height="10"/></svg>',
      );
      fireEvent.click(getIconButton(getPartListItem("Part 1"), "close"));
      await waitFor(() => expect(screen.queryByText("Part 1")).toBeNull());
      expect(
        getPartListItem("Whole drawing").classList.contains("lyra--selected"),
      ).toBe(true);
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
