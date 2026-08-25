// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("Range.useAddToNewPlot", () => {
  it("should favorite the retrieved ranges into the slice", async () => {
    const a = await createTestRange(client);
    const b = await createTestRange(client);
    const { wrapper, store } = await createConsoleWrapper({ client });
    const { result } = renderHook(() => Range.useAddToNewPlot(), { wrapper });
    act(() => result.current([a.key, b.key]));
    await waitFor(() => {
      const keys = Session.Range.selectKeys(store.getState());
      expect(keys).toContain(a.key);
      expect(keys).toContain(b.key);
    });
  });
});
