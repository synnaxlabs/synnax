// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import { renderSchematic } from "@/feature/schematic/testutil";
import { getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { captureBrowserDownloads, getIconButton, uniqueName } from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Schematic.Toolbar", () => {
  describe("when the schematic is not editable", () => {
    it("prompts to enable editing and flips the editable state on click", async () => {
      const name = uniqueName("toolbar");
      const { key, store } = await renderSchematic(Schematic.Toolbar, {
        schematic: { name },
      });
      await screen.findByText(`${name} is not editable. To make changes,`);
      fireEvent.click(await screen.findByText("enable editing."));
      await waitFor(() =>
        expect(
          Session.Schematic.selectState({ state: store.getState(), key }).editable,
        ).toBe(true),
      );
      expect(await screen.findByText("Search symbols")).toBeDefined();
    });

    it("flips the editable state from the release-control prompt when control is acquired", async () => {
      const name = uniqueName("toolbar");
      const { key, store } = await renderSchematic(Schematic.Toolbar, {
        schematic: { name },
        sessionState: {
          control: { authority: 1, status: "acquired" },
        },
      });
      fireEvent.click(await screen.findByText("release control and enable editing."));
      await waitFor(() =>
        expect(
          Session.Schematic.selectState({ state: store.getState(), key }).editable,
        ).toBe(true),
      );
    });
  });

  describe("when the schematic is editable", () => {
    it("switches to the properties tab and shows the empty selection hint", async () => {
      const { key, store } = await renderSchematic(Schematic.Toolbar, {
        sessionState: { editable: true },
      });
      fireEvent.click(await screen.findByText("Properties"));
      await waitFor(() =>
        expect(
          Session.Schematic.selectToolbar({ state: store.getState(), key }).selectedTab,
        ).toBe("properties"),
      );
      expect(
        await screen.findByText(
          "Select a schematic element to configure its properties.",
        ),
      ).toBeDefined();
    });

    it("switches to the control tab and updates the legend visibility", async () => {
      const { key, store } = await renderSchematic(Schematic.Toolbar, {
        sessionState: { editable: true },
      });
      fireEvent.click(await screen.findByText("Control"));
      await screen.findByText("Control authority");
      const switchInput = getSwitch("Show control state legend");
      expect(
        Session.Schematic.selectLegend({ state: store.getState(), key }).visible,
      ).toBe(true);
      fireEvent.click(switchInput);
      await waitFor(() =>
        expect(
          Session.Schematic.selectLegend({ state: store.getState(), key }).visible,
        ).toBe(false),
      );
    });
  });

  describe("export", () => {
    it("downloads the schematic as a typed JSON export", async () => {
      const downloads = captureBrowserDownloads();
      const name = uniqueName("export");
      const { result } = await renderSchematic(Schematic.Toolbar, {
        schematic: { name },
        sessionState: { editable: true },
      });
      await screen.findByText(name);
      fireEvent.click(getIconButton(result.baseElement, "export"));
      await waitFor(() => expect(downloads.anchors).toHaveLength(1));
      expect(downloads.anchors[0].download).toBe(`${name}.json`);
      const contents = JSON.parse(
        new TextDecoder().decode(await downloads.blobs[0].arrayBuffer()),
      );
      // The Core serializes the schematic into the flat imex envelope: numeric version,
      // and the resource key is dropped.
      expect(contents).toMatchObject({ name, type: "schematic", version: 6 });
    });
  });
});
