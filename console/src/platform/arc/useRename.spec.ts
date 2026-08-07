// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { List } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/platform/arc";
import { client } from "@/platform/arc/testutil";
import { createConsoleWrapper } from "@/testutil";

const getItemFor = (item: arc.Arc): List.GetItem<arc.Key, arc.Arc> =>
  List.createGetItem<arc.Key, arc.Arc>(
    (key) => (key === item.key ? item : undefined),
    (keys) => keys.filter((key) => key === item.key).map(() => item),
  );

describe("arc useRename", () => {
  it("should persist the rename to the cluster", async () => {
    const original = await client.arcs.create({
      name: id.create(),
      mode: "graph",
      graph: { nodes: [], edges: [] },
    });
    const newName = id.create();
    const { wrapper } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => Arc.useRename(getItemFor(original)), {
      wrapper,
    });

    act(() => {
      result.current.update({ key: original.key, name: newName });
    });

    await waitFor(async () => {
      const retrieved = await client.arcs.retrieve(original.key);
      expect(retrieved.name).toBe(newName);
    });
  });
});
