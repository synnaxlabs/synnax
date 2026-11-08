// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import { type record, zod } from "@synnaxlabs/x";
import { assert, describe, expect, it, vi } from "vitest";
import z from "zod";

import { type framer } from "@/framer";
import { MultipleFoundError, NotFoundError, ValidationError } from "@/index";
import { query } from "@/query";

interface Thing extends record.Keyed<string> {
  key: string;
  name: string;
  size: number;
}

const thing = (key: string, size = 1): Thing => ({ key, name: `${key}-name`, size });

const requestZ = z.strictObject({
  keys: z.string().array().optional(),
  where: z
    .object({
      key: z
        .object({ eq: z.string().optional(), in: z.string().array().optional() })
        .optional(),
      name: z
        .object({ eq: z.string().optional(), in: z.string().array().optional() })
        .optional(),
    })
    .optional(),
  minSize: z.number().optional(),
  searchTerm: z.string().optional(),
});
interface Request extends z.infer<typeof requestZ> {}

const paramsZ = z.union([
  requestZ,
  query.keyListZ(z.string()),
  z
    .strictObject({
      key: z.union([z.string(), z.string().array()]),
      minSize: z.number().optional(),
    })
    .transform(({ key, minSize }): Request => ({
      where: { key: Array.isArray(key) ? { in: key } : { eq: key } },
      minSize,
    })),
]);

// A live but silent stream: subscribed answers stay maintained, so reads
// serve from the cache without a network fetch.
const silentStreamer = (): framer.Streamer => {
  let closed = false;
  const streamer: framer.Streamer = {
    keys: [],
    update: async () => {},
    close: () => {
      closed = true;
    },
    next: async () => {
      // Blocks until closed so the read loop stays alive like a real stream.
      while (!closed) await new Promise((resolve) => setTimeout(resolve, 5));
      return { done: true, value: undefined };
    },
    read: async () => {
      await streamer.next();
      throw new EOF();
    },
    [Symbol.asyncIterator]: () => streamer,
  };
  return streamer;
};

const newCache = () => new query.Cache({ openStreamer: async () => silentStreamer() });

class Client extends query.Retriever<typeof paramsZ, string, Thing> {
  constructor(
    cache: query.Cache,
    fetchKeys: (keys: string[]) => Promise<Thing[]>,
    fetchRequest: (req: Request) => Promise<Thing[]>,
  ) {
    const store = cache.createTable<string, Thing>({
      name: "things",
      fetch: fetchKeys,
    });
    super(cache, {
      name: "thing",
      table: store,
      request: {
        schema: paramsZ,
        fetch: async (req) => await fetchRequest(req),
        matches: (r, req) => {
          const keys = "keys" in req ? req.keys : undefined;
          if (keys != null && !keys.includes(r.key)) return false;
          const minSize = "minSize" in req ? req.minSize : undefined;
          return minSize == null || r.size >= minSize;
        },
      },
    });
    this.store = store;
  }

  readonly store: query.Table<string, Thing>;
}

const expectDeleted = <D extends query.Data>(
  value: query.Cached<D> | undefined,
): query.Deleted<D> => {
  if (!query.Deleted.matches<D>(value)) throw new Error("expected a deleted answer");
  return value;
};

describe("Retriever", () => {
  describe("derived single space", () => {
    it("resolves a { key } param to one record through the table fetch", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      const result = await client.retrieve({ key: "a" });
      expect(result).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledWith(["a"]);
    });

    it("resolves a bare key as { key } shorthand through the same query", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      const result = await client.retrieve("a");
      expect(result).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      expect(client.watch({ key: "a" }).getCached()).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledTimes(1);
    });

    it("resolves a bare key array as { keys } shorthand", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      const result = await client.retrieve(["a", "b"]);
      expect(result).toEqual([thing("a"), thing("b")]);
      expect(fetchKeys).toHaveBeenCalledWith(["a", "b"]);
    });

    it("omits a missing key from a bare key array read as { keys }", async () => {
      const client = new Client(
        newCache(),
        async () => [thing("a")],
        async () => [],
      );
      expect(await client.retrieve(["a", "b"])).toEqual([thing("a")]);
    });

    it("gives a bare array the meaning the schema assigns it", async () => {
      const namesZ = z.preprocess(
        (p) => (Array.isArray(p) ? { names: p } : p),
        z.object({
          keys: z.string().array().optional(),
          names: z.string().array().optional(),
        }),
      );
      class NamesClient extends query.Retriever<typeof namesZ, string, Thing> {
        constructor(
          cache: query.Cache,
          fetchRequest: (req: z.output<typeof namesZ>) => Promise<Thing[]>,
        ) {
          const store = cache.createTable<string, Thing>({
            name: "things",
            fetch: async (keys) => keys.map((k) => thing(k)),
          });
          super(cache, {
            name: "thing",
            table: store,
            request: { schema: namesZ, fetch: async (req) => await fetchRequest(req) },
          });
        }
      }
      const fetchRequest = vi.fn(async (_req: z.output<typeof namesZ>) => [thing("a")]);
      const client = new NamesClient(newCache(), fetchRequest);
      const result = await client.retrieve(["a-name"]);
      expect(result).toEqual([thing("a")]);
      expect(fetchRequest.mock.calls[0][0]).toEqual({ names: ["a-name"] });
    });

    it("rejects a bare key array the request schema does not accept", async () => {
      const keylessZ = z.object({ minSize: z.number().optional() });
      class KeylessClient extends query.Retriever<typeof keylessZ, string, Thing> {
        constructor(cache: query.Cache) {
          const store = cache.createTable<string, Thing>({
            name: "things",
            fetch: async (keys) => keys.map((k) => thing(k)),
          });
          super(cache, {
            name: "thing",
            table: store,
            request: { schema: keylessZ, fetch: async () => [] },
          });
        }
      }
      const client = new KeylessClient(newCache());
      await expect(client.retrieve(["a"])).rejects.toThrow(zod.ParseError);
    });

    it("serves a cached entry without a network fetch", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      client.store.set([thing("a")]);
      const result = await client.retrieve({ key: "a" });
      expect(result).toEqual(thing("a"));
      expect(fetchKeys).not.toHaveBeenCalled();
    });

    it("rejects with NotFoundError when the key does not exist", async () => {
      const client = new Client(
        newCache(),
        async () => [],
        async () => [],
      );
      await expect(client.retrieve({ key: "missing" })).rejects.toThrow(NotFoundError);
    });

    it("routes a list on key to the request space", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const fetchRequest = vi.fn(async () => []);
      const client = new Client(newCache(), fetchKeys, fetchRequest);
      const result = await client.retrieve({ key: ["a", "b"] });
      expect(result).toEqual([thing("a"), thing("b")]);
      expect(fetchKeys).toHaveBeenCalledWith(["a", "b"]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("returns one record for a plain key beside other fields", async () => {
      const fetchRequest = vi.fn(async () => [thing("a", 3)]);
      const client = new Client(newCache(), async () => [], fetchRequest);
      const result: Thing = await client.retrieve({ key: "a", minSize: 2 });
      expect(result).toEqual(thing("a", 3));
      expect(fetchRequest).toHaveBeenCalledWith({
        where: { key: { eq: "a" } },
        minSize: 2,
      });
    });

    it("throws a NotFoundError when a plain key beside other fields matches nothing", async () => {
      const client = new Client(
        newCache(),
        async () => [],
        async () => [],
      );
      await expect(client.retrieve({ key: "a", minSize: 2 })).rejects.toThrow(
        new NotFoundError('Found no thing where key = "a"'),
      );
    });

    it("returns one record for a query that sets one", async () => {
      const fetchRequest = vi.fn(async () => [thing("a", 3)]);
      const client = new Client(newCache(), async () => [], fetchRequest);
      const result: Thing = await client.retrieve({ minSize: 2, one: true });
      expect(result).toEqual(thing("a", 3));
      expect(fetchRequest).toHaveBeenCalledWith({ minSize: 2 });
    });

    it("throws a MultipleFoundError when a query that sets one matches two", async () => {
      const client = new Client(
        newCache(),
        async () => [],
        async () => [thing("a", 3), thing("b", 4)],
      );
      await expect(client.retrieve({ minSize: 2, one: true })).rejects.toThrow(
        new MultipleFoundError("Expected one thing, but found 2"),
      );
    });

    it("routes a plain key that sets one through the single space", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const fetchRequest = vi.fn(async () => []);
      const client = new Client(newCache(), fetchKeys, fetchRequest);
      expect(await client.retrieve({ key: "a", one: true })).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it.each([
      ["a limit", { limit: 1 }],
      ["a cursor", { after: { key: "a" } }],
    ])("rejects a query that sets one beside %s", async (_, page) => {
      const fetchRequest = vi.fn(async () => []);
      const client = new Client(newCache(), async () => [], fetchRequest);
      await expect(
        client.retrieve({ minSize: 2, one: true, ...page } as never),
      ).rejects.toThrow(
        new ValidationError(
          "a query for one record takes no limit or cursor, since a page could hide a " +
            "second match",
        ),
      );
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("delivers deletion through onChange and flips retrieve to not-found", async () => {
      const client = new Client(
        newCache(),
        async (keys) => keys.map((k) => thing(k)),
        async () => [],
      );
      await client.retrieve({ key: "a" });
      const handler = vi.fn();
      const stop = client.watch({ key: "a" }).onChange(handler);
      client.store.delete("a");
      expect(expectDeleted(handler.mock.lastCall?.[0]).corpse).toEqual(thing("a"));
      await expect(client.retrieve({ key: "a" })).rejects.toThrow(NotFoundError);
      stop();
    });

    it("answers getCached from the table for never-queried keys", async () => {
      const client = new Client(
        newCache(),
        async () => [],
        async () => [],
      );
      client.store.set([thing("a")]);
      expect(client.watch({ key: "a" }).getCached()).toEqual(thing("a"));
    });
  });

  describe("request space", () => {
    it("routes keys-only requests through the table fetch", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const fetchRequest = vi.fn(async () => []);
      const client = new Client(newCache(), fetchKeys, fetchRequest);
      client.store.set([thing("a")]);
      const results = await client.retrieve({ keys: ["a", "b"] });
      expect(results.map((t) => t.key)).toEqual(["a", "b"]);
      expect(fetchKeys).toHaveBeenCalledWith(["b"]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("routes non-keys requests through the request fetch and hydrates", async () => {
      const fetchRequest = vi.fn(async (req: Request) =>
        [thing("a", 5), thing("b", 9)].filter((t) => t.size >= (req.minSize ?? 0)),
      );
      const client = new Client(newCache(), async () => [], fetchRequest);
      const results = await client.retrieve({ minSize: 6 });
      expect(results).toEqual([thing("b", 9)]);
      expect(client.store.get("b")).toEqual(thing("b", 9));
    });

    it("keeps a write that landed while a request fetch was in flight", async () => {
      let release: (things: Thing[]) => void = () => {};
      const fetchRequest = vi.fn(
        async () => await new Promise<Thing[]>((resolve) => (release = resolve)),
      );
      const client = new Client(newCache(), async () => [], fetchRequest);
      const pending = client.retrieve({ minSize: 1 });
      await vi.waitFor(() => expect(fetchRequest).toHaveBeenCalledTimes(1));
      client.store.set([{ ...thing("a"), name: "renamed" }]);
      release([thing("a")]);
      await pending;
      expect(client.store.get("a")?.name).toEqual("renamed");
    });

    it("routes a where that only matches a key list through the table fetch", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const fetchRequest = vi.fn(async () => []);
      const client = new Client(newCache(), fetchKeys, fetchRequest);
      client.store.set([thing("a")]);
      const results = await client.retrieve({ where: { key: { in: ["a", "b"] } } });
      expect(results.map((t) => t.key)).toEqual(["a", "b"]);
      expect(fetchKeys).toHaveBeenCalledWith(["b"]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("answers a where with an empty key list from the table", async () => {
      const fetchRequest = vi.fn(async () => []);
      const client = new Client(newCache(), async () => [], fetchRequest);
      expect(await client.retrieve({ where: { key: { in: [] } } })).toEqual([]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    describe("keysRequestZ", () => {
      const keysZ = z.union([query.keysRequestZ(z.string()), requestZ]);
      class KeysClient extends query.Retriever<typeof keysZ, string, Thing> {
        constructor(
          cache: query.Cache,
          fetchKeys: (keys: string[]) => Promise<Thing[]>,
          fetchRequest: () => Promise<Thing[]>,
        ) {
          const store = cache.createTable<string, Thing>({
            name: "things",
            fetch: fetchKeys,
          });
          super(cache, {
            name: "thing",
            table: store,
            request: { schema: keysZ, fetch: fetchRequest },
          });
          this.store = store;
        }

        readonly store: query.Table<string, Thing>;
      }

      it("routes a bare key list through the table fetch", async () => {
        const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
        const fetchRequest = vi.fn(async () => []);
        const client = new KeysClient(newCache(), fetchKeys, fetchRequest);
        client.store.set([thing("a")]);
        const results = await client.retrieve(["a", "b"]);
        expect(results.map((t) => t.key)).toEqual(["a", "b"]);
        expect(fetchKeys).toHaveBeenCalledWith(["b"]);
        expect(fetchRequest).not.toHaveBeenCalled();
      });

      it("answers an empty key list with no fetch", async () => {
        const fetchKeys = vi.fn(async () => []);
        const fetchRequest = vi.fn(async () => []);
        const client = new KeysClient(newCache(), fetchKeys, fetchRequest);
        expect(await client.retrieve([])).toEqual([]);
        expect(fetchKeys).not.toHaveBeenCalled();
        expect(fetchRequest).not.toHaveBeenCalled();
      });

      it("returns each record of a bare key list once, in key order", async () => {
        const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
        const client = new KeysClient(newCache(), fetchKeys, async () => []);
        const results = await client.retrieve(["b", "a", "b"]);
        expect(results.map((t) => t.key)).toEqual(["b", "a"]);
        expect(client.watch(["b", "a", "b"]).getCached()).toEqual(results);
      });

      it("names a repeated missing key of a bare key list once", async () => {
        const client = new KeysClient(newCache(), async () => [thing("a")], vi.fn());
        await expect(client.retrieve(["b", "a", "b"])).rejects.toThrow(
          new NotFoundError("thing with key b not found"),
        );
      });

      it("throws a NotFoundError naming the missing key of a bare key list", async () => {
        const client = new KeysClient(newCache(), async () => [thing("a")], vi.fn());
        await expect(client.retrieve(["a", "b"])).rejects.toThrow(
          new NotFoundError("thing with key b not found"),
        );
      });

      it("throws a NotFoundError naming every missing key of a bare key list", async () => {
        const client = new KeysClient(newCache(), async () => [thing("a")], vi.fn());
        await expect(client.retrieve(["b", "a", "c"])).rejects.toThrow(
          new NotFoundError("thing with keys b, c not found"),
        );
      });

      it("returns the records that exist for a filter on keys", async () => {
        const client = new KeysClient(newCache(), async () => [thing("a")], vi.fn());
        const results = await client.retrieve({ where: { key: { in: ["a", "b"] } } });
        expect(results.map((t) => t.key)).toEqual(["a"]);
      });
    });

    it("routes a where with another line through the request fetch", async () => {
      const fetchRequest = vi.fn(async () => [thing("a")]);
      const client = new Client(newCache(), async () => [], fetchRequest);
      await client.retrieve({ where: { key: { in: ["a"] }, name: { eq: "a-name" } } });
      await client.retrieve({ where: { key: { eq: "a", in: ["a"] } } });
      expect(fetchRequest).toHaveBeenCalledTimes(2);
    });

    it("treats a request with keys plus another set field as non-keys-only", async () => {
      const fetchRequest = vi.fn(async () => [thing("a", 9)]);
      const client = new Client(newCache(), async () => [], fetchRequest);
      await client.retrieve({ keys: ["a"], minSize: 5 });
      expect(fetchRequest).toHaveBeenCalledTimes(1);
    });

    it("answers a keys-only request from records cached by other queries", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      await client.retrieve("a");
      await client.retrieve("b");
      expect(await client.retrieve({ keys: ["a", "b"] })).toEqual([
        thing("a"),
        thing("b"),
      ]);
      expect(fetchKeys).toHaveBeenCalledTimes(2);
    });

    it("matches the fetch answer for duplicated and deleted keys", async () => {
      const existing = new Set(["a", "b"]);
      const fetchKeys = vi.fn(async (keys: string[]) =>
        keys.filter((k) => existing.has(k)).map((k) => thing(k)),
      );
      const client = new Client(newCache(), fetchKeys, async () => []);
      await client.retrieve(["a", "b"]);
      existing.delete("b");
      client.store.delete("b");
      const cached = client.watch(["a", "a", "b"]).getCached();
      expect(cached).toEqual([thing("a")]);
      expect(await client.retrieve({ keys: ["a", "a", "b"] })).toEqual(cached);
    });
  });

  describe("normalization memoization", () => {
    it("validates a single params object once across repeat reads", async () => {
      const singleZ = z.strictObject({ key: z.string() }).transform(({ key }) => key);
      class SingleClient extends query.Retriever<
        typeof requestZ,
        string,
        Thing,
        Thing,
        { key: string }
      > {
        constructor(cache: query.Cache) {
          const store = cache.createTable<string, Thing>({
            name: "things",
            fetch: async (keys) => keys.map((k) => thing(k)),
          });
          const single = cache.queries<string, Thing, string, Thing>({
            name: "thing",
            table: store,
            fetch: async (key) => (await store.retrieve([key])).map((r) => r.key),
            compose: (records) => records[0],
            single: true,
          });
          super(cache, {
            name: "thing",
            table: store,
            request: { schema: requestZ, fetch: async () => [] },
            single: { schema: singleZ, space: single },
          });
        }
      }
      const safeParse = vi.spyOn(singleZ, "safeParse");
      const client = new SingleClient(newCache());
      const params = { key: "a" };
      await client.retrieve(params);
      const stop = client.watch(params).onChange(vi.fn());
      expect(client.watch(params).getCached()).toEqual(thing("a"));
      client.watch(params).getCached();
      expect(safeParse).toHaveBeenCalledTimes(1);
      stop();
      safeParse.mockRestore();
    });
  });

  describe("compose", () => {
    interface Sugared extends Thing {
      sugared: true;
    }

    class SugaredClient extends query.Retriever<
      typeof requestZ,
      string,
      Thing,
      Sugared
    > {
      constructor(cache: query.Cache) {
        const store = cache.createTable<string, Thing>({
          name: "things",
          fetch: async (keys) => keys.map((k) => thing(k)),
        });
        super(cache, {
          name: "thing",
          table: store,
          request: { schema: requestZ, fetch: async () => [thing("a")] },
          compose: (record) => ({ ...record, sugared: true }),
        });
      }
    }

    it("applies the per-record compose to single and request answers", async () => {
      const client = new SugaredClient(newCache());
      const single = await client.retrieve({ key: "a" });
      expect(single.sugared).toBe(true);
      const many = await client.retrieve({});
      expect(many.every((t) => t.sugared)).toBe(true);
    });
  });

  describe("custom single space", () => {
    type NameOrKey = string;

    class ByNameClient extends query.Retriever<
      typeof requestZ,
      string,
      Thing,
      Thing,
      NameOrKey
    > {
      constructor(cache: query.Cache, fetchKeys: (keys: string[]) => Promise<Thing[]>) {
        const store = cache.createTable<string, Thing>({
          name: "things",
          fetch: fetchKeys,
        });
        const single = cache.queries<string, Thing, string, Thing>({
          name: "thing",
          table: store,
          fetch: async (name) => {
            const records = await fetchKeys([name.replace("name:", "")]);
            store.ingest(records);
            return records.map((r) => r.key);
          },
          compose: (records) => records[0],
          single: true,
        });
        super(cache, {
          name: "thing",
          table: store,
          request: { schema: requestZ, fetch: async () => [] },
          single: { schema: z.string(), space: single },
        });
      }
    }

    it("routes single params through the custom space", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new ByNameClient(newCache(), fetchKeys);
      const result = await client.retrieve("name:a");
      expect(result).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledWith(["a"]);
    });

    describe("malformed single params", () => {
      class StrictSingleClient extends query.Retriever<
        typeof paramsZ,
        string,
        Thing,
        Thing,
        { key: string }
      > {
        constructor(
          cache: query.Cache,
          fetchKeys: (keys: string[]) => Promise<Thing[]>,
          fetchRequest: (req: Request) => Promise<Thing[]>,
        ) {
          const store = cache.createTable<string, Thing>({
            name: "things",
            fetch: fetchKeys,
          });
          const single = cache.queries<string, Thing, string, Thing>({
            name: "thing",
            table: store,
            fetch: async (key) => {
              const records = await fetchKeys([key]);
              store.ingest(records);
              return records.map((r) => r.key);
            },
            compose: (records) => records[0],
            single: true,
          });
          super(cache, {
            name: "thing",
            table: store,
            request: {
              schema: paramsZ,
              fetch: async (req) => await fetchRequest(req),
            },
            single: {
              schema: z
                .union([
                  z.strictObject({ key: z.string() }),
                  z.string().transform((key) => ({ key })),
                ])
                .transform(({ key }) => key),
              space: single,
            },
          });
        }
      }

      it("rejects key-bearing params that fail the single schema", async () => {
        const fetchRequest = vi.fn(async () => []);
        const client = new StrictSingleClient(
          newCache(),
          async (keys) => keys.map((k) => thing(k)),
          fetchRequest,
        );
        const err: unknown = await client
          .retrieve({ key: "a", detailsSchema: undefined } as never)
          .then(() => null)
          .catch((exc: unknown) => exc);
        assert(err instanceof ValidationError);
        expect(err.message).toBe(
          "thing params address a single key but do not match the single-query " +
            "schema.\nFailed to parse thing single query params (1 issue)\n\n  {\n" +
            '    "key": "a",\n  ✗ "detailsSchema": "[undefined]"  × unexpected key\n  }',
        );
        expect(err.cause).toBeInstanceOf(zod.ParseError);
        expect(fetchRequest).not.toHaveBeenCalled();
      });

      it("routes a list on key past the single schema", async () => {
        const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
        const fetchRequest = vi.fn(async () => []);
        const client = new StrictSingleClient(newCache(), fetchKeys, fetchRequest);
        const result = await client.retrieve({ key: ["a"] });
        expect(result).toEqual([thing("a")]);
        expect(fetchRequest).not.toHaveBeenCalled();
      });

      it("routes a clean { key } param through the single space", async () => {
        const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
        const client = new StrictSingleClient(newCache(), fetchKeys, async () => []);
        const result = await client.retrieve({ key: "a" });
        expect(result).toEqual(thing("a"));
        expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      });

      it("resolves a bare key through the schema's own bare-key arm", async () => {
        const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
        const client = new StrictSingleClient(newCache(), fetchKeys, async () => []);
        const result = await client.retrieve("a");
        expect(result).toEqual(thing("a"));
        expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      });

      it("rejects a bare key when the single query needs more than a key", async () => {
        class IDSingleClient extends query.Retriever<
          typeof requestZ,
          string,
          Thing,
          Thing,
          { type: string; key: string }
        > {
          constructor(
            cache: query.Cache,
            fetchKeys: (keys: string[]) => Promise<Thing[]>,
            fetchRequest: (req: Request) => Promise<Thing[]>,
          ) {
            const store = cache.createTable<string, Thing>({
              name: "things",
              fetch: fetchKeys,
            });
            const single = cache.queries<string, Thing, string, Thing>({
              name: "thing",
              table: store,
              fetch: async (key) => {
                const records = await fetchKeys([key]);
                store.ingest(records);
                return records.map((r) => r.key);
              },
              compose: (records) => records[0],
              single: true,
            });
            super(cache, {
              name: "thing",
              table: store,
              request: {
                schema: requestZ,
                fetch: async (req) => await fetchRequest(req),
              },
              single: {
                schema: z
                  .strictObject({ type: z.string(), key: z.string() })
                  .transform(({ key }) => key),
                space: single,
              },
            });
          }
        }
        const fetchRequest = vi.fn(async () => []);
        const client = new IDSingleClient(
          newCache(),
          async (keys) => keys.map((k) => thing(k)),
          fetchRequest,
        );
        const err: unknown = await client
          .retrieve("a")
          .then(() => null)
          .catch((exc: unknown) => exc);
        assert(err instanceof ValidationError);
        expect(err.message).toBe(
          "thing params address a single key but do not match the single-query " +
            "schema.\nFailed to parse thing single query params (1 issue)\n\n" +
            '  × expected object, received "a"',
        );
        expect(fetchRequest).not.toHaveBeenCalled();
      });
    });
  });

  describe("shortcut", () => {
    const keyZ = z.string().regex(/^[a-z]$/);
    const shortcutZ = z.union([
      keyZ.transform((key) => ({ where: { key: { eq: key } } })),
      z.string().transform((name) => ({ where: { name: { eq: name } } })),
      keyZ.array().transform((keys) => ({ where: { key: { in: keys } } })),
      z
        .string()
        .array()
        .transform((names) => ({ where: { name: { in: names } } })),
    ]);

    class NamedClient extends query.Retriever<typeof requestZ, string, Thing> {
      constructor(
        cache: query.Cache,
        fetchKeys: (keys: string[]) => Promise<Thing[]>,
        fetchRequest: (req: Request) => Promise<Thing[]>,
      ) {
        const store = cache.createTable<string, Thing>({
          name: "things",
          fetch: fetchKeys,
        });
        super(cache, {
          name: "thing",
          table: store,
          request: { schema: requestZ, fetch: async (req) => await fetchRequest(req) },
          shortcut: shortcutZ,
        });
      }
    }

    const byName =
      (...records: Thing[]) =>
      async (req: Request): Promise<Thing[]> => {
        const names = req.where?.name?.in ?? [req.where?.name?.eq];
        return records.filter((r) => names.includes(r.name));
      };

    const byKey = async (keys: string[]): Promise<Thing[]> => keys.map((k) => thing(k));

    it("returns the one record a bare name addresses", async () => {
      const fetchRequest = vi.fn(byName(thing("a"), thing("b")));
      const client = new NamedClient(newCache(), byKey, fetchRequest);
      expect(await client.retrieve("a-name")).toEqual(thing("a"));
      expect(fetchRequest).toHaveBeenCalledWith({ where: { name: { eq: "a-name" } } });
    });

    it("answers getCached for a bare name it has read", async () => {
      const fetchRequest = vi.fn(byName(thing("a")));
      const client = new NamedClient(newCache(), byKey, fetchRequest);
      await client.retrieve("a-name");
      expect(client.watch("a-name").getCached()).toEqual(thing("a"));
      expect(fetchRequest).toHaveBeenCalledTimes(1);
    });

    it("routes a bare key through the single space", async () => {
      const fetchKeys = vi.fn(byKey);
      const fetchRequest = vi.fn(byName());
      const client = new NamedClient(newCache(), fetchKeys, fetchRequest);
      expect(await client.retrieve("a")).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("returns the records a list of names addresses", async () => {
      const fetchRequest = vi.fn(byName(thing("a"), thing("b"), thing("c")));
      const client = new NamedClient(newCache(), byKey, fetchRequest);
      expect(await client.retrieve(["a-name", "b-name"])).toEqual([
        thing("a"),
        thing("b"),
      ]);
      expect(fetchRequest).toHaveBeenCalledWith({
        where: { name: { in: ["a-name", "b-name"] } },
      });
    });

    it("returns each record of a bare key list once, in key order", async () => {
      const fetchKeys = vi.fn(byKey);
      const fetchRequest = vi.fn(byName());
      const client = new NamedClient(newCache(), fetchKeys, fetchRequest);
      expect(await client.retrieve(["b", "a", "b"])).toEqual([thing("b"), thing("a")]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("reads a list as names when a member is not a key", async () => {
      const fetchRequest = vi.fn(byName(thing("b")));
      const client = new NamedClient(newCache(), byKey, fetchRequest);
      expect(await client.retrieve(["a", "b-name"])).toEqual([thing("b")]);
      expect(fetchRequest).toHaveBeenCalledWith({
        where: { name: { in: ["a", "b-name"] } },
      });
    });

    it("throws a NotFoundError when no record has the name", async () => {
      const client = new NamedClient(newCache(), byKey, byName(thing("a")));
      await expect(client.retrieve("c-name")).rejects.toThrow(
        new NotFoundError('Found no thing where name = "c-name"'),
      );
    });

    it("throws a MultipleFoundError when two records have the name", async () => {
      const dup = (key: string): Thing => ({ ...thing(key), name: "dup" });
      const client = new NamedClient(newCache(), byKey, byName(dup("a"), dup("b")));
      await expect(client.retrieve("dup")).rejects.toThrow(
        new MultipleFoundError('Expected one thing where name = "dup", but found 2'),
      );
    });
  });

  describe("list", () => {
    const listRequestZ = z.strictObject({
      keys: z.string().array().optional(),
      minSize: z.number().optional(),
      limit: z.number().optional(),
      order: z.object({ field: z.literal("name") }).optional(),
      after: z.object({ key: z.string(), name: z.string().optional() }).optional(),
    });
    type ListRequest = z.infer<typeof listRequestZ>;
    type Cursor = NonNullable<ListRequest["after"]>;

    class ListClient extends query.Retriever<typeof listRequestZ, string, Thing> {
      constructor(
        cache: query.Cache,
        fetchRequest: (req: ListRequest) => Promise<Thing[]>,
      ) {
        const store = cache.createTable<string, Thing>({ name: "things" });
        super(cache, {
          name: "thing",
          table: store,
          request: {
            schema: listRequestZ,
            fetch: async (req) => await fetchRequest(req),
            matches: (r, req) =>
              (req.keys == null || req.keys.includes(r.key)) &&
              (req.minSize == null || r.size >= req.minSize),
            list: {
              cursor: (r, order): Cursor =>
                order == null ? { key: r.key } : { key: r.key, name: r.name },
              compare: (a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0),
            },
          },
        });
        this.store = store;
      }

      readonly store: query.Table<string, Thing>;
    }

    it("should list through the request fetch", async () => {
      const fetchRequest = vi.fn(async () => [thing("a", 3), thing("b", 4)]);
      const client = new ListClient(newCache(), fetchRequest);
      const window = await client
        .watch({ minSize: 2, order: { field: "name" }, limit: 2 })
        .retrieve();
      expect(window.records).toEqual([thing("a", 3), thing("b", 4)]);
      expect(window.end).toEqual({ key: "b", name: "b-name" });
      expect(fetchRequest).toHaveBeenCalledWith({
        minSize: 2,
        order: { field: "name" },
        limit: 2,
      });
    });

    it("should answer a keys query from the table with no fetch", async () => {
      const fetchRequest = vi.fn(async () => []);
      const client = new ListClient(newCache(), fetchRequest);
      client.store.set([thing("b"), thing("a")]);
      const list = client.watch({ keys: ["b", "a"] });
      expect(list.getCached()?.records).toEqual([thing("a"), thing("b")]);
      expect((await list.retrieve()).records).toEqual([thing("a"), thing("b")]);
      expect(fetchRequest).not.toHaveBeenCalled();
    });

    it("should not answer a keys query with an unknown key from the table", () => {
      const client = new ListClient(newCache(), async () => []);
      client.store.set(thing("a"));
      expect(client.watch({ keys: ["a", "c"] }).getCached()).toBeUndefined();
    });

    it("should leave a deleted key out of a keys query answer", () => {
      const client = new ListClient(newCache(), async () => []);
      client.store.set([thing("a"), thing("b")]);
      client.store.delete("b");
      expect(client.watch({ keys: ["a", "b"] }).getCached()?.records).toEqual([
        thing("a"),
      ]);
    });

    it("should return one list for equal queries", () => {
      const client = new ListClient(newCache(), async () => []);
      const list = client.watch({ minSize: 3 });
      expect(client.watch({ minSize: 3 })).toBe(list);
      expect(client.watch({ minSize: 3, order: undefined })).toBe(list);
    });

    it("should validate a params object once across repeat reads", () => {
      const safeParse = vi.spyOn(listRequestZ, "safeParse");
      const client = new ListClient(newCache(), async () => []);
      const params = { minSize: 3 };
      client.watch(params);
      client.watch(params);
      expect(safeParse).toHaveBeenCalledTimes(1);
      safeParse.mockRestore();
    });

    it("should throw for params the request schema rejects", () => {
      const client = new ListClient(newCache(), async () => []);
      expect(() => client.watch({ bogus: 1 } as never)).toThrow(zod.ParseError);
    });

    it("should throw for a domain with no lists", () => {
      const client = new Client(
        newCache(),
        async () => [],
        async () => [],
      );
      expect(() => client.watch({})).toThrow("thing has no lists");
    });
  });
});
