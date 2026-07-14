// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { Schematic as PSchematic, type Status, Theming } from "@synnaxlabs/pluto";
import { location, uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import { client, renderSchematic } from "@/feature/schematic/testutil";
import { findButton } from "@/platform/modals/testutil";
import { type Session } from "@/session";
import {
  CaptureStatuses,
  getIconButton,
  getInputByItemLabel,
  isPlutoDisabled,
} from "@/testutil";

const theme = Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxDark);

const createValveConfig = (): Record<string, unknown> =>
  PSchematic.Node.resolveSpec("valve").defaultConfig(theme);

interface RenderPropertiesParams {
  nodeKeys: string[];
  sessionState?: Partial<Session.Schematic.State>;
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
  createConfig?: () => Record<string, unknown>;
}

const renderProperties = async ({
  nodeKeys,
  sessionState,
  onStatuses = () => {},
  createConfig = createValveConfig,
}: RenderPropertiesParams) => {
  const configs: Record<string, unknown> = {};
  nodeKeys.forEach((key) => (configs[key] = createConfig()));
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

const retrieveConfig = async (
  key: string,
  nodeKey: string,
): Promise<Record<string, unknown>> => {
  const retrieved = await client.schematics.retrieve({ key });
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
      await screen.findByText("Style");
      const input = getInputByItemLabel(result.container, "Label");
      fireEvent.change(input, { target: { value: "Main Valve" } });
      await waitFor(async () => {
        const config = await retrieveConfig(key, "n1");
        expect(config.label).toMatchObject({ label: "Main Valve" });
      });
      expect(await screen.findByText("Main Valve")).toBeDefined();
    });
  });

  describe("missing custom symbol", () => {
    it("shows the missing symbol form when the referenced spec does not exist", async () => {
      await renderProperties({
        nodeKeys: ["n1"],
        createConfig: () => ({
          ...(PSchematic.Node.resolveSpec("customStatic").defaultConfig(
            theme,
          ) as Record<string, unknown>),
          specKey: uuid.create(),
        }),
      });
      expect(
        await screen.findByText(
          "The custom symbol referenced by this node was not found.",
        ),
      ).toBeDefined();
      expect(screen.getByText("Or create a new symbol in:")).toBeDefined();
      expect(isPlutoDisabled(findButton("Create New Symbol"))).toBe(true);
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
      const input = getInputByItemLabel(result.container, "Label wrap width");
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
