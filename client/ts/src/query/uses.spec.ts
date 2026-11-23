// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { Series } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { framer } from "@/framer";
import { MissingCredentialsError } from "@/index";
import { query } from "@/query";
import { PushStreamer } from "@/testutil";

interface Rec {
  key: string;
  uses?: unknown[];
}

type Request = { keys: string[]; includeUses?: boolean };

describe("Uses", () => {
  it.each([
    ["cannot reach the Core", () => new Unreachable()],
    ["runs after a logout", () => new MissingCredentialsError()],
  ])("should not report a refetch that %s", async (_, offline) => {
    const onError = vi.fn();
    const stream = new PushStreamer();
    const cache = new query.Cache({ openStreamer: async () => stream, onError });
    const retrieve = vi
      .fn<(request: Request) => Promise<Rec[]>>()
      .mockRejectedValueOnce(offline())
      .mockRejectedValueOnce(new Error("boom"));
    const uses = new query.Uses<string, Rec, Request>({
      name: "recs",
      cache,
      table: cache.createTable<string, Rec>({ name: "recs" }),
      key: z.string(),
      retrieve,
      request: (key) => ({ keys: "in" in key ? key.in : [key.eq], includeUses: true }),
      channels: ["recs_changes"],
    });
    uses.created(["a"]);
    uses.trigger.attach(
      { keys: ["a"], includeUses: true },
      () => {},
      () => ["a"],
    );
    await cache.ensureStreaming();
    await expect.poll(() => retrieve.mock.calls.length).toBe(1);
    // The second refetch fails in a way that reports, so the first has settled.
    stream.push(new framer.Frame({ recs_changes: new Series([1]) }));
    await expect.poll(() => onError.mock.calls.length).toBe(1);
    expect(onError.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        message: "failed to refetch recs uses",
        cause: expect.objectContaining({ message: "boom" }),
      }),
    );
    await cache.close();
  });
});
