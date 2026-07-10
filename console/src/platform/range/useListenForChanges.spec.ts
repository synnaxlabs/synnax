// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const createRange = async (): Promise<ranger.Range> => await createTestRange(client);

const preloadedFor = (range: ranger.Range) => ({
  [Session.Range.SLICE_NAME]: {
    version: 0 as const,
    ranges: Session.Range.fromClient(range),
  },
});

describe("Range.useListenForChanges", () => {
  it("should update a favorited range in the slice when it is renamed remotely", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor(range),
    });
    renderHook(() => Range.useListenForChanges(), { wrapper });
    const nextName = id.create();
    await client.ranges.rename(range.key, nextName);
    await waitFor(() => {
      expect(Session.Range.selectState(store.getState(), range.key)?.name).toBe(
        nextName,
      );
    });
  });

  it("should remove a favorited range from the slice when it is deleted remotely", async () => {
    const range = await createRange();
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: preloadedFor(range),
    });
    renderHook(() => Range.useListenForChanges(), { wrapper });
    expect(Session.Range.selectState(store.getState(), range.key)).toBeDefined();
    await client.ranges.delete(range.key);
    await waitFor(() => {
      expect(Session.Range.selectState(store.getState(), range.key)).toBeUndefined();
    });
  });
});
