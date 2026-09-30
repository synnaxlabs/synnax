// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type Status } from "@synnaxlabs/lyra/status";
import { Schematic as PSchematic } from "@synnaxlabs/pluto";
import { color, location, uuid } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, renderSchematic } from "@/feature/schematic/testutil";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import {
  assertDefined,
  CaptureStatuses,
  getIconButton,
  getInputByItemLabel,
  isPlutoDisabled,
} from "@/testutil";

const createValveConfig = (): Record<string, unknown> =>
  PSchematic.Node.createConfig({ variant: "valve" });

interface RenderPropertiesParams {
  nodeKeys: string[];
  sessionState?: Partial<Session.Schematic.State>;
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
  createConfig?: (key: string) => Record<string, unknown>;
}

const renderProperties = async ({
  nodeKeys,
  sessionState,
  onStatuses = () => {},
  createConfig = createValveConfig,
}: RenderPropertiesParams) => {
  const configs: Record<string, unknown> = {};
  nodeKeys.forEach((key) => (configs[key] = createConfig(key)));
  const Harness = (): ReactElement => (
    <>
      <Schematic.Toolbar />
      <CaptureStatuses onStatuses={onStatuses} />
    </>
  );
  return await renderSchematic(Harness, {
    schematic: {
      nodes: nodeKeys.map((key) => ({ key, position: { x: 0, y: 0 } })),
      configs: configs as schematic.New["configs"],
    },
    sessionState: {
      editable: true,
      selected: nodeKeys,
      toolbar: { selectedTab: "properties", selectedSymbolGroup: "general" },
      ...sessionState,
    },
  });
};

const createBoxConfig = (): Record<string, unknown> =>
  PSchematic.Node.createConfig({ variant: "box" });

const RED = "#ff0000";
const GREEN = "#00ff00";
const BLUE = "#0000ff";

// The role control's hex box, labeled with the role name.
const roleInput = (label: string): HTMLInputElement => {
  const input = screen
    .getAllByLabelText(label)
    .find((el): el is HTMLInputElement => el instanceof HTMLInputElement);
  assertDefined(input);
  return input;
};

const setRoleColor = (label: string, hex: string): void => {
  fireEvent.change(roleInput(label), { target: { value: hex.slice(1) } });
};

// Picks Auto in the role control's picker.
const clearRoleColor = (label: string): void => {
  const swatch = roleInput(label)
    .closest(".pluto-color-input")
    ?.querySelector<HTMLElement>(".pluto-color-swatch");
  assertDefined(swatch);
  fireEvent.click(swatch);
  fireEvent.click(screen.getByLabelText("Auto"));
};

const selectionSwatches = (): HTMLElement[] => {
  const item = screen.getByText("Selection colors").closest(".pluto-input__item");
  assertDefined(item);
  return Array.from(item.querySelectorAll<HTMLElement>(".pluto-color-swatch"));
};

const hexOf = (value: unknown): string | undefined =>
  value == null ? undefined : color.hex(value as color.Crude);

const retrieveConfig = async (
  key: string,
  nodeKey: string,
): Promise<Record<string, unknown>> => {
  const retrieved = await client.schematics.retrieve(key);
  return retrieved.configs[nodeKey];
};

const pollBothOrientations = async (key: string, rotated: string): Promise<void> => {
  await expect
    .poll(async () => {
      const [n1, n2] = await Promise.all([
        retrieveConfig(key, "n1"),
        retrieveConfig(key, "n2"),
      ]);
      return { n1: n1.orientation, n2: n2.orientation };
    })
    .toEqual({ n1: rotated, n2: rotated });
};

describe("Schematic toolbar Properties", () => {
  describe("single selection", () => {
    it("writes label edits back to the schematic config and the breadcrumb", async () => {
      const { key, result } = await renderProperties({ nodeKeys: ["n1"] });
      fireEvent.click(await screen.findByRole("tab", { name: "Style" }));
      const input = getInputByItemLabel(result.container, "Label");
      fireEvent.change(input, { target: { value: "Main Valve" } });
      await waitFor(async () => {
        const config = await retrieveConfig(key, "n1");
        expect(config.label).toMatchObject({ label: "Main Valve" });
      });
      expect(await screen.findByText("Main Valve")).toBeDefined();
    });
  });

  describe("tabs", () => {
    it("opens a symbol on its control tab", async () => {
      const { result } = await renderProperties({ nodeKeys: ["n1"] });
      await screen.findByRole("tab", { name: "Style" });
      const rail = result.container.querySelector<HTMLElement>(
        ".pluto-properties-tabs",
      );
      assertDefined(rail);
      const tab = within(rail).getByRole("tab", { name: "Control" });
      expect(tab.ariaSelected).toBe("true");
    });

    it("keeps the selected tab when another symbol is selected", async () => {
      const { key, store } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        sessionState: { selected: ["n1"] },
      });
      fireEvent.click(await screen.findByRole("tab", { name: "Style" }));
      act(() => {
        store.dispatch(Session.Schematic.setSelected({ key, selected: ["n2"] }));
      });
      await waitFor(() =>
        expect(screen.getByRole("tab", { name: "Style" }).ariaSelected).toBe("true"),
      );
    });
  });

  describe("missing custom symbol", () => {
    it("shows the missing symbol form when the referenced spec does not exist", async () => {
      await renderProperties({
        nodeKeys: ["n1"],
        createConfig: () =>
          PSchematic.Node.createConfig({
            variant: "custom_static",
            specKey: uuid.create(),
          }),
      });
      expect(
        await screen.findByText(
          "The custom symbol referenced by this node was not found.",
        ),
      ).toBeDefined();
      expect(screen.getByText("Or create a symbol in:")).toBeDefined();
      expect(isPlutoDisabled(findButton("Create symbol"))).toBe(true);
    });
  });

  describe("grouped selection", () => {
    const nodeKeys = ["g1", "n1", "n2", "n3"];
    const createGroupedConfig = (key: string): Record<string, unknown> =>
      key === "g1"
        ? {
            ...(PSchematic.Node.createConfig({
              variant: "group_box",
            }) as unknown as Record<string, unknown>),
            members: ["n1", "n2"],
          }
        : createValveConfig();

    it("routes a grouped member plus a loose symbol to the multi form", async () => {
      await renderProperties({
        nodeKeys,
        createConfig: createGroupedConfig,
        sessionState: { selected: ["n1", "n3"] },
      });
      await screen.findByText("Align");
    });

    const createLockedConfig = (key: string): Record<string, unknown> =>
      key === "g1"
        ? { ...createGroupedConfig(key), locked: true }
        : createGroupedConfig(key);

    it("skips a locked group's members when rotating", async () => {
      const { key, result } = await renderProperties({
        nodeKeys,
        createConfig: createLockedConfig,
        sessionState: { selected: nodeKeys },
      });
      await screen.findByText("Align");
      fireEvent.click(getIconButton(result.container, "rotate-group-cw"));
      await expect
        .poll(async () => (await retrieveConfig(key, "n3")).orientation)
        .toEqual(location.rotate("left", "clockwise"));
      expect((await retrieveConfig(key, "n1")).orientation).toBe("left");
    });

    it("skips a locked group's members when applying label props", async () => {
      const { key, result } = await renderProperties({
        nodeKeys,
        createConfig: createLockedConfig,
        sessionState: { selected: nodeKeys },
      });
      await screen.findByText("Align");
      const input = getInputByItemLabel(result.container, "Wrap width");
      fireEvent.change(input, { target: { value: "200" } });
      fireEvent.blur(input);
      await expect
        .poll(async () => (await retrieveConfig(key, "n3")).label)
        .toMatchObject({ maxInlineSize: 200 });
      expect((await retrieveConfig(key, "n1")).label).not.toMatchObject({
        maxInlineSize: 200,
      });
    });

    it("skips a locked group's members when setting a role color", async () => {
      const { key } = await renderProperties({
        nodeKeys,
        createConfig: createLockedConfig,
        sessionState: { selected: nodeKeys },
      });
      await screen.findByText("Colors");
      setRoleColor("Stroke", RED);
      await expect
        .poll(async () => hexOf((await retrieveConfig(key, "n3")).strokeColor))
        .toBe(RED);
      expect((await retrieveConfig(key, "n1")).strokeColor).toBeUndefined();
    });

    it("routes a group selected with its members to the multi form", async () => {
      await renderProperties({
        nodeKeys,
        createConfig: createGroupedConfig,
        sessionState: { selected: ["g1", "n1", "n2"] },
      });
      await screen.findByText("Align");
      expect(screen.queryByText("Groups have no editable properties.")).toBeNull();
    });
  });

  describe("multi-element colors", () => {
    it("sets a stroke on every selected symbol whose stroke is auto", async () => {
      const { key } = await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Colors");
      expect(roleInput("Stroke").placeholder).toBe("Auto");
      setRoleColor("Stroke", RED);
      await expect
        .poll(async () =>
          (
            await Promise.all([retrieveConfig(key, "n1"), retrieveConfig(key, "n2")])
          ).map((c) => hexOf(c.strokeColor)),
        )
        .toEqual([RED, RED]);
    });

    it("shows Mixed when the selected strokes differ", async () => {
      await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) =>
          key === "n1"
            ? { ...createValveConfig(), strokeColor: RED }
            : createValveConfig(),
      });
      await screen.findByText("Colors");
      expect(roleInput("Stroke").placeholder).toBe("Mixed");
      expect(roleInput("Stroke").value).toBe("");
    });

    it("shows the shared stroke when every selected symbol has it", async () => {
      await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: () => ({ ...createValveConfig(), strokeColor: RED }),
      });
      await screen.findByText("Colors");
      expect(roleInput("Stroke").value).toBe("ff0000");
    });

    it("clears the stroke on every selected symbol with Auto", async () => {
      const { key } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) => ({
          ...createValveConfig(),
          strokeColor: key === "n1" ? RED : GREEN,
        }),
      });
      await screen.findByText("Colors");
      clearRoleColor("Stroke");
      await expect
        .poll(async () =>
          (
            await Promise.all([retrieveConfig(key, "n1"), retrieveConfig(key, "n2")])
          ).map((c) => c.strokeColor),
        )
        .toEqual([undefined, undefined]);
    });

    it("shows the fill that auto-filled symbols paint", async () => {
      await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: () => PSchematic.Node.createConfig({ variant: "button" }),
      });
      await screen.findByText("Colors");
      expect(roleInput("Fill").placeholder).toBe("Auto");
      const swatch = roleInput("Fill")
        .closest(".pluto-color-input")
        ?.querySelector<HTMLElement>(".pluto-color-swatch");
      assertDefined(swatch);
      expect(swatch.style.getPropertyValue("--pluto-swatch-color")).not.toBe(
        "rgba(0, 0, 0, 0)",
      );
    });

    it("shows Mixed when auto-filled symbols paint different fills", async () => {
      await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) =>
          key === "n1"
            ? PSchematic.Node.createConfig({ variant: "button" })
            : createBoxConfig(),
      });
      await screen.findByText("Colors");
      expect(roleInput("Fill").placeholder).toBe("Mixed");
    });

    it("writes a fill only to the symbols that have one", async () => {
      const { key } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) => (key === "n1" ? createBoxConfig() : createValveConfig()),
      });
      await screen.findByText("Colors");
      setRoleColor("Fill", BLUE);
      await expect
        .poll(async () => hexOf((await retrieveConfig(key, "n1")).fillColor))
        .toBe(BLUE);
      expect((await retrieveConfig(key, "n2")).fillColor).toBeUndefined();
    });

    it("hides the roles that no selected symbol has", async () => {
      await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Colors");
      expect(screen.queryByLabelText("Fill")).toBeNull();
      expect(screen.queryByLabelText("Text")).toBeNull();
    });

    it("lists only stored colors in Selection colors", async () => {
      await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Colors");
      expect(screen.queryByText("Selection colors")).toBeNull();
    });

    it("recolors every field that holds a selection color", async () => {
      const { key } = await renderProperties({
        nodeKeys: ["n1", "n2", "n3"],
        createConfig: (key) => {
          if (key === "n1") return { ...createBoxConfig(), strokeColor: RED };
          if (key === "n2") return { ...createValveConfig(), stalenessColor: RED };
          return { ...createBoxConfig(), fillColor: GREEN };
        },
      });
      await screen.findByText("Selection colors");
      const [red] = selectionSwatches();
      expect(selectionSwatches()).toHaveLength(2);
      fireEvent.click(red);
      fireEvent.change(screen.getByLabelText("Hex"), {
        target: { value: BLUE.slice(1) },
      });
      fireEvent.keyDown(document.body, { code: "Escape" });
      await expect
        .poll(async () => {
          const [n1, n2, n3] = await Promise.all(
            ["n1", "n2", "n3"].map(async (k) => await retrieveConfig(key, k)),
          );
          return [hexOf(n1.strokeColor), hexOf(n2.stalenessColor), hexOf(n3.fillColor)];
        })
        .toEqual([BLUE, BLUE, GREEN]);
    });

    it("recolors live while the picker is open without merging groups", async () => {
      const { key } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) =>
          key === "n1"
            ? { ...createBoxConfig(), strokeColor: RED }
            : { ...createBoxConfig(), strokeColor: GREEN },
      });
      await screen.findByText("Selection colors");
      const [red] = selectionSwatches();
      const strokes = async (): Promise<Array<string | undefined>> =>
        await Promise.all(
          ["n1", "n2"].map(async (k) =>
            hexOf((await retrieveConfig(key, k)).strokeColor),
          ),
        );
      fireEvent.click(red);
      const hex = screen.getByLabelText("Hex");
      fireEvent.change(hex, { target: { value: GREEN.slice(1) } });
      await expect.poll(strokes).toEqual([GREEN, GREEN]);
      fireEvent.change(hex, { target: { value: BLUE.slice(1) } });
      await expect.poll(strokes).toEqual([BLUE, GREEN]);
    });

    it("recolors a state option through Selection colors", async () => {
      const { key } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) =>
          key === "n1"
            ? {
                ...PSchematic.Node.createConfig({ variant: "state_indicator" }),
                options: [{ key: "a", name: "A", value: 0, color: RED }],
              }
            : createValveConfig(),
      });
      await screen.findByText("Selection colors");
      const [red] = selectionSwatches();
      fireEvent.click(red);
      fireEvent.change(screen.getByLabelText("Hex"), {
        target: { value: BLUE.slice(1) },
      });
      fireEvent.keyDown(document.body, { code: "Escape" });
      await expect
        .poll(async () => {
          const { options } = (await retrieveConfig(key, "n1")) as {
            options: Array<{ color?: unknown }>;
          };
          return hexOf(options[0].color);
        })
        .toBe(BLUE);
    });

    it("sets a staleness color only on the symbols that have staleness", async () => {
      const { key } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        createConfig: (key) => (key === "n1" ? createValveConfig() : createBoxConfig()),
      });
      await screen.findByText("Staleness");
      setRoleColor("Color", RED);
      await expect
        .poll(async () => hexOf((await retrieveConfig(key, "n1")).stalenessColor))
        .toBe(RED);
      expect((await retrieveConfig(key, "n2")).stalenessColor).toBeUndefined();
    });

    it("hides the staleness group when no selected symbol has staleness", async () => {
      await renderProperties({ nodeKeys: ["n1", "n2"], createConfig: createBoxConfig });
      await screen.findByText("Colors");
      expect(screen.queryByText("Staleness")).toBeNull();
    });

    it("applies a symbol scale to every selected symbol", async () => {
      const { key, result } = await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Symbol size");
      const input = getInputByItemLabel(result.container, "Scale");
      fireEvent.change(input, { target: { value: "150" } });
      fireEvent.blur(input);
      await expect
        .poll(async () =>
          (
            await Promise.all([retrieveConfig(key, "n1"), retrieveConfig(key, "n2")])
          ).map((c) => c.scale),
        )
        .toEqual([1.5, 1.5]);
    });

    it("hides the symbol size group when no selected symbol has a scale", async () => {
      await renderProperties({ nodeKeys: ["n1", "n2"], createConfig: createBoxConfig });
      await screen.findByText("Colors");
      expect(screen.queryByText("Symbol size")).toBeNull();
    });
  });

  describe("multiple selection", () => {
    it("hides the spacing controls for fewer than three selections", async () => {
      await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Align");
      expect(screen.queryByText("Spacing")).toBeNull();
    });

    it("rotates every selected symbol's orientation", async () => {
      const { key, result } = await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Align");
      fireEvent.click(getIconButton(result.container, "rotate-group-cw"));
      await pollBothOrientations(key, location.rotate("left", "clockwise"));
    });

    it("applies label wrap width to every selected symbol", async () => {
      const { key, result } = await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Align");
      const input = getInputByItemLabel(result.container, "Wrap width");
      fireEvent.change(input, { target: { value: "200" } });
      fireEvent.blur(input);
      await expect
        .poll(async () => (await retrieveConfig(key, "n1")).label)
        .toMatchObject({ maxInlineSize: 200 });
    });

    it("rotates the whole selection, still rotating orientations when layout fails", async () => {
      const { key, result } = await renderProperties({ nodeKeys: ["n1", "n2"] });
      await screen.findByText("Align");
      fireEvent.click(getIconButton(result.container, "rotate-around-center-ccw"));
      await pollBothOrientations(key, location.rotate("left", "counterclockwise"));
    });

    it("reports a layout error for every alignment action in an unmeasurable DOM", async () => {
      const statuses: Status.NotificationSpec[] = [];
      const onStatuses = (next: Status.NotificationSpec[]) => {
        statuses.length = 0;
        statuses.push(...next);
      };
      const { key, result } = await renderProperties({
        nodeKeys: ["n1", "n2"],
        onStatuses,
      });
      await screen.findByText("Align");
      for (const icon of [
        "align-y-center",
        "align-x-center",
        "align-left",
        "align-top",
        "align-bottom",
        "align-right",
      ])
        fireEvent.click(getIconButton(result.container, icon));
      await waitFor(() =>
        expect(
          statuses.some(
            (st) =>
              st.variant === "error" &&
              st.message === "failed to calculate schematic node layout",
          ),
        ).toBe(true),
      );
      expect((await retrieveConfig(key, "n1")).orientation).toBe("left");
    });

    it("surfaces a layout error when node elements are not measurable", async () => {
      const statuses: Status.NotificationSpec[] = [];
      const onStatuses = vi.fn((next: Status.NotificationSpec[]) => {
        statuses.length = 0;
        statuses.push(...next);
      });
      const { result } = await renderProperties({
        nodeKeys: ["n1", "n2", "n3"],
        onStatuses,
      });
      await screen.findByText("Spacing");
      fireEvent.click(getIconButton(result.container, "distribute-x"));
      await waitFor(() =>
        expect(
          statuses.some(
            (st) =>
              st.variant === "error" &&
              st.message === "failed to calculate schematic node layout",
          ),
        ).toBe(true),
      );
    });
  });
});
