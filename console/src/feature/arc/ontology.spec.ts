// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, createTestClient } from "@synnaxlabs/client";
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
  it("should retrieve the arc and open it as a tab on select", async () => {
    const arc = await client.arcs.create({
      name: uniqueName("arc"),
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const openTab = vi.fn();
    const store = await createTestStore();
    const id = clientArc.ontologyID(arc.key);
    onSelect({
      ...createBaseProps({
        client,
        store,
        overrides: { openTab, handleError: createExecutingHandleError() },
      }),
      selection: [createResource(id, arc.name)],
    });
    await waitFor(() => expect(openTab).toHaveBeenCalledTimes(1));
    expect(openTab).toHaveBeenCalledWith({ resource: id });
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
