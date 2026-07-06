// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ontology, ranger } from "@synnaxlabs/client";
import { List, Select } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/feature/range";
import { type Layout } from "@/platform/layout";
import { createBaseProps, createResource } from "@/platform/ontology/testutil";
import { Range as CommonRange } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createTestStore, renderWithConsole } from "@/testutil";

const client = createTestClient();

describe("range/ontology", () => {
  describe("onSelect", () => {
    it("adds the range to the session slice and places its overview", async () => {
      const rng = await createTestRange(client);
      const store = await createTestStore();
      const placeLayout = vi.fn((_: Layout.PlacerArgs) => ({
        windowKey: "",
        key: "",
      }));
      Range.ONTOLOGY_SERVICE.onSelect?.({
        ...createBaseProps({
          client,
          store,
          overrides: {
            placeLayout,
            handleError: (exc) => {
              if (typeof exc === "function") void exc();
            },
          },
        }),
        selection: [createResource(ranger.ontologyID(rng.key), rng.name)],
      });
      await waitFor(() => {
        expect(Session.Range.selectState(store.getState(), rng.key)?.name).toBe(
          rng.name,
        );
      });
      await waitFor(() => expect(placeLayout).toHaveBeenCalledTimes(1));
      const [layout] = placeLayout.mock.calls[0];
      if (typeof layout === "function") throw new Error("expected a layout object");
      expect(layout.key).toBe(rng.key);
      expect(layout.name).toBe(rng.name);
      expect(layout.type).toBe(CommonRange.OVERVIEW_LAYOUT_TYPE);
    });
  });

  describe("haulItems", () => {
    it("returns a range haul item carrying the resource payload", async () => {
      const rng = await createTestRange(client);
      const resource = createResource(ranger.ontologyID(rng.key), rng.name, {
        key: rng.key,
        name: rng.name,
        timeRange: rng.timeRange,
      });
      const items = Range.ONTOLOGY_SERVICE.haulItems(resource);
      expect(items).toHaveLength(1);
      expect(items[0].key).toBe(rng.key);
    });

    it("returns nothing when the resource has no data payload", () => {
      const resource = createResource(ranger.ontologyID("some-key"), "no-data");
      expect(Range.ONTOLOGY_SERVICE.haulItems(resource)).toHaveLength(0);
    });
  });

  describe("PaletteListItem", () => {
    it("renders the resource name and time range", async () => {
      const rng = await createTestRange(client);
      const resource = createResource(ranger.ontologyID(rng.key), rng.name, {
        timeRange: rng.timeRange.numeric,
      });
      const PaletteListItem = Range.ONTOLOGY_SERVICE.PaletteListItem;
      if (PaletteListItem == null) throw new Error("PaletteListItem is not defined");
      const Harness = (): ReactElement => {
        const staticProps = List.useStaticData<string, ontology.Resource>({
          data: [resource],
        });
        return (
          <Select.Frame<string, ontology.Resource>
            {...staticProps}
            value={undefined}
            onChange={() => {}}
          >
            <PaletteListItem key={resource.key} itemKey={resource.key} index={0} />
          </Select.Frame>
        );
      };
      const { container } = await renderWithConsole(<Harness />);
      expect(await screen.findByText(rng.name)).toBeTruthy();
      // createTestRange builds a range starting now, which formats as "Today ...".
      expect(container.textContent).toContain("Today");
    });
  });
});
