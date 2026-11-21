// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { Series, TimeSpan, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { type z } from "zod";

import { framer } from "@/framer";
import { query } from "@/query";
import { secret } from "@/secret";
import { PushStreamer } from "@/testutil";

interface RetrieveRequest {
  where: { key: { eq?: secret.Key; in?: secret.Key[] } };
  includeUses?: boolean;
}

const PAST_DEBOUNCE = TimeSpan.seconds(1).milliseconds;

/**
 * Creates a secret client on a fake transport. The next `core.failUses` retrieves that
 * ask for uses fail with an error that is not a connection error.
 */
const createFake = async () => {
  const usesRequests: secret.Key[][] = [];
  const core = { failUses: 0 };
  const send = async (
    target: string,
    req: RetrieveRequest,
    _: z.ZodType,
    res: z.ZodType,
  ) => {
    if (target !== "/secret/retrieve")
      throw new Error(`unexpected request to ${target}`);
    const { eq, in: keys = eq == null ? [] : [eq] } = req.where.key;
    if (req.includeUses === true) {
      usesRequests.push(keys);
      if (core.failUses > 0) {
        core.failUses--;
        throw new Error("internal server error");
      }
    }
    return res.parse({
      secrets: keys.map((key) => ({
        key,
        name: key,
        source: "stored",
        updated: 0,
        ...(req.includeUses === true && { uses: [{ type: "certificate", key }] }),
      })),
    });
  };
  const unary = { use: () => {}, send } as unknown as UnaryClient;
  const streamer = new PushStreamer();
  const cache = new query.Cache({
    openStreamer: async () => streamer,
    breaker: { baseInterval: TimeSpan.milliseconds(1) },
    onError: () => {},
  });
  onTestFinished(async () => await cache.close());
  const secrets = new secret.Client({ unary, cache });
  await cache.ensureStreaming();
  const change = (...keys: string[]): void =>
    streamer.push(
      new framer.Frame({
        [query.REFERENCE_CHANGE_CHANNEL_NAME]: new Series(
          keys.map((key) => `${secret.TYPE_RESOURCE_ID.type}:${key}`),
        ),
      }),
    );
  return { secrets, core, usesRequests, change };
};

describe("Secret uses refetch after a failure", () => {
  it("should fetch the uses of a failed refetch again at the next change", async () => {
    vi.useFakeTimers({
      toFake: ["setTimeout", "clearTimeout", "Date", "Temporal"],
      shouldAdvanceTime: true,
    });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    const { secrets, core, usesRequests, change } = await createFake();
    const [failed, later] = [uuid.create(), uuid.create()];
    for (const key of [failed, later]) {
      const q = { key, includeUses: true };
      onTestFinished(secrets.watch(q).onChange(() => {}));
      await secrets.retrieve(q);
    }
    core.failUses = 1;
    change(failed);
    await vi.advanceTimersByTimeAsync(PAST_DEBOUNCE);
    expect(usesRequests).toEqual([[failed], [later], [failed]]);
    change(later);
    await vi.advanceTimersByTimeAsync(PAST_DEBOUNCE);
    expect(usesRequests.at(-1)).toEqual(expect.arrayContaining([failed, later]));
  });
});
