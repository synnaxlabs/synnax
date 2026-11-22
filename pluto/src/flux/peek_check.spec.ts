// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const Wrapper = createSynnaxWrapper({ client });

beforeAll(async () => {
  await waitFor(() => expect(client.connection.status.details.epoch).toBe(1));
});

describe("Checker peek", () => {
  it("should not answer undefined after a fetch lands for a query with getCached", async () => {
    const cache = new Map<string, number>();
    const { peek, useResult } = Flux.createRetrieve<{ key: string }, number>({
      name: "Cached",
      retrieve: async ({ query: { key } }) => {
        cache.set(key, 7);
        return 7;
      },
      getCached: ({ query: { key } }) => cache.get(key),
    });
    const { result } = renderHook(() => useResult({ key: "a" }), { wrapper: Wrapper });
    await waitFor(() => expect(result.current.variant).toEqual("success"));
    expect(result.current.data).toEqual(7);
    // Either answer is sound: the settled value, or a loud refusal as the selectors
    // give for a missing getCached. A silent undefined reads as "not yet fetched".
    let outcome: unknown;
    try {
      outcome = peek(client, { key: "a" });
    } catch (e) {
      outcome = e instanceof UnexpectedError ? "UnexpectedError" : e;
    }
    expect([7, "UnexpectedError"]).toContain(outcome);
  });
});
