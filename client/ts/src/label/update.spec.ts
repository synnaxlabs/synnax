// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { patch } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { label } from "@/label";
import { query } from "@/query";

const A = "00000000-0000-4000-8000-00000000000a";
const B = "00000000-0000-4000-8000-00000000000b";

interface Request {
  target: string;
  req: unknown;
}

// A Core that holds two labels and answers each write with respond.
const createClient = async (respond: (r: Request) => unknown = () => ({})) => {
  const requests: Request[] = [];
  const unary: UnaryClient = {
    use: () => {},
    send: async (target, req, _, resZ) => {
      if (target === "/label/retrieve")
        return resZ.parse({
          labels: [
            { key: A, name: "Alpha", color: "#ff0000" },
            { key: B, name: "Beta", color: "#00ff00" },
          ],
        });
      requests.push({ target, req });
      return resZ.parse(await respond({ target, req }));
    },
  };
  // With no change stream, a reply settles each update.
  const cache = new query.Cache({ openStreamer: null, onError: () => {} });
  onTestFinished(async () => await cache.close());
  const labels = new label.Client({ unary, cache, relabel: {} });
  await labels.retrieve([A, B]);
  const cachedName = (key: label.Key): string | undefined => {
    const cached = labels.watch(key).getCached();
    return query.isLive(cached) ? cached.name : undefined;
  };
  return { labels, requests, cachedName };
};

const updateAction = (value: label.Update) => ({
  type: "update",
  update: { value, guards: [] },
});

describe("label.Client.update", () => {
  it("should send one update through the dispatch endpoint", async () => {
    let shown: string | undefined;
    const { labels, requests, cachedName } = await createClient(() => {
      shown = cachedName(A);
      return {};
    });
    await labels.update({ key: A, name: "Gamma" });
    expect(requests).toEqual([
      {
        target: "/label/dispatch",
        req: {
          key: A,
          dispatchKey: expect.any(String),
          actions: [updateAction({ name: "Gamma" })],
        },
      },
    ]);
    expect(shown).toBe("Gamma");
    expect(cachedName(A)).toBe("Gamma");
  });

  it("should send several updates as one batch", async () => {
    let shown: Array<string | undefined> = [];
    const { labels, requests, cachedName } = await createClient(() => {
      shown = [cachedName(A), cachedName(B)];
      return { responses: [{}, {}] };
    });
    await labels.update([
      { key: A, name: "Gamma" },
      { key: B, name: "Delta" },
    ]);
    expect(requests).toEqual([
      {
        target: "/batch",
        req: {
          requests: [A, B].map((key, i) => ({
            endpoint: "/label/dispatch",
            payload: {
              key,
              dispatchKey: expect.any(String),
              actions: [updateAction({ name: ["Gamma", "Delta"][i] })],
            },
          })),
        },
      },
    ]);
    expect(shown).toEqual(["Gamma", "Delta"]);
    expect([cachedName(A), cachedName(B)]).toEqual(["Gamma", "Delta"]);
  });

  it("should roll back an update that the Core finds stale", async () => {
    const { labels, cachedName } = await createClient(() => {
      throw new patch.StaleError("Alpha is not a label");
    });
    await expect(labels.update({ key: A, name: "Gamma" })).rejects.toThrow(
      patch.StaleError,
    );
    expect(cachedName(A)).toBe("Alpha");
  });

  it("should roll back every update of a batch that the Core finds stale", async () => {
    const { labels, cachedName } = await createClient(() => {
      throw new patch.StaleError("Beta is not a label");
    });
    await expect(
      labels.update([
        { key: A, name: "Gamma" },
        { key: B, name: "Delta" },
      ]),
    ).rejects.toThrow(patch.StaleError);
    expect([cachedName(A), cachedName(B)]).toEqual(["Alpha", "Beta"]);
  });
});
