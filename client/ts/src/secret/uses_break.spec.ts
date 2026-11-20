// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Timeout, type UnaryClient } from "@synnaxlabs/freighter";
import { Series, TimeSpan, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished, vi } from "vitest";
import { type z } from "zod";

import { framer } from "@/framer";
import { query } from "@/query";
import { secret } from "@/secret";
import { expectLive, PushStreamer } from "@/testutil";

const usesOf = (key: secret.Key) => [{ type: "certificate", key }];
const movedUsesOf = (key: secret.Key) => [{ type: "certificate", key: `${key}-moved` }];

interface RetrieveRequest {
  where: { key: { eq?: secret.Key; in?: secret.Key[] } };
  includeUses?: boolean;
}

const PAST_DEBOUNCE = TimeSpan.seconds(1).milliseconds;

describe("Secret uses refetch", () => {
  it("should refetch the uses of a change whose refetch timed out", async () => {
    vi.useFakeTimers({
      toFake: ["setTimeout", "clearTimeout", "Date", "Temporal"],
      shouldAdvanceTime: true,
    });
    onTestFinished(() => {
      vi.useRealTimers();
    });
    const usesRequests: secret.Key[][] = [];
    const core = { usesOf, fail: null as Error | null };
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
        if (core.fail != null) throw core.fail;
      }
      return res.parse({
        secrets: keys.map((key) => ({
          key,
          name: key,
          source: "stored",
          updated: 0,
          ...(req.includeUses === true && { uses: core.usesOf(key) }),
        })),
      });
    };
    const unary = { use: () => {}, send } as unknown as UnaryClient;
    const stream = new PushStreamer();
    const cache = new query.Cache({
      openStreamer: async () => stream,
      breaker: { baseInterval: TimeSpan.milliseconds(1) },
      onError: () => {},
    });
    onTestFinished(async () => await cache.close());
    const secrets = new secret.Client({ unary, cache });
    await cache.ensureStreaming();
    const change = (key: secret.Key): void =>
      stream.push(
        new framer.Frame({
          [query.REFERENCE_CHANGE_CHANNEL_NAME]: new Series([
            `${secret.TYPE_RESOURCE_ID.type}:${key}`,
          ]),
        }),
      );
    const [slow, later] = [uuid.create(), uuid.create()];
    for (const key of [slow, later]) {
      const q = { key, includeUses: true };
      onTestFinished(secrets.watch(q).onChange(() => {}));
      await secrets.retrieve(q);
    }
    expect(usesRequests).toEqual([[slow], [later]]);
    // A task moves its secret, and the refetch that the change starts times out. The
    // change stream stays open, so no reconnect follows.
    core.usesOf = movedUsesOf;
    core.fail = new Timeout("the Core answered too late");
    change(slow);
    await vi.advanceTimersByTimeAsync(PAST_DEBOUNCE);
    expect(usesRequests).toEqual([[slow], [later], [slow]]);
    core.fail = null;
    change(later);
    await vi.advanceTimersByTimeAsync(PAST_DEBOUNCE);
    const cached = expectLive(
      secrets.watch({ key: slow, includeUses: true }).getCached(),
    );
    expect(cached.uses).toEqual(movedUsesOf(slow));
  });
});
