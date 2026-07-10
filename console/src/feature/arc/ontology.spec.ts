// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Arc } from "@/feature/arc";
import {
  createBaseProps,
  createExecutingHandleError,
  createResource,
} from "@/platform/ontology/testutil";
import { assertDefined, createTestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const { onSelect } = Arc.ONTOLOGY_SERVICE;
assertDefined(onSelect, "arc ontology service has no onSelect");

describe("arc ontology", () => {
  it("should place the arc editor layout on select", async () => {
    const arc = await client.arcs.create({
      name: uniqueName("arc"),
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const placeLayout = vi.fn();
    const store = await createTestStore();
    onSelect({
      ...createBaseProps({
        client,
        store,
        overrides: { placeLayout, handleError: createExecutingHandleError() },
      }),
      selection: [createResource(clientArc.ontologyID(arc.key), arc.name)],
    });
    await waitFor(() => expect(placeLayout).toHaveBeenCalledTimes(1));
    const placed = placeLayout.mock.calls[0][0]({ dispatch: store.dispatch });
    expect(placed.key).toBe(arc.key);
    expect(placed.name).toBe(arc.name);
    expect(placed.type).toBe(Arc.EDITOR_LAYOUT_TYPE);
  });

  it("should report an error when the arc cannot be loaded", async () => {
    const errors: string[] = [];
    const store = await createTestStore();
    const ghost = createResource(
      clientArc.ontologyID("00000000-0000-0000-0000-000000000000"),
      "Ghost Arc",
    );
    onSelect({
      ...createBaseProps({
        client,
        store,
        overrides: {
          handleError: createExecutingHandleError((message) => errors.push(message)),
        },
      }),
      selection: [ghost],
    });
    await waitFor(() => expect(errors).toContain("Failed to load Ghost Arc"));
  });
});
