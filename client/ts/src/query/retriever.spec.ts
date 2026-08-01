// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import z from "zod";

import { NotFoundError, ValidationError } from "@/errors";
import { query } from "@/query";
import { Deleted } from "@/query/deleted";

interface Thing extends record.Keyed<string> {
  key: string;
  name: string;
  size: number;
}

const thing = (key: string, size = 1): Thing => ({ key, name: `${key}-name`, size });

const requestZ = z.object({
  keys: z.string().array().optional(),
  minSize: z.number().optional(),
  searchTerm: z.string().optional(),
});
interface Request extends z.infer<typeof requestZ> {}

const newCache = () => new query.Cache({ openStreamer: null });

class Client extends query.Retriever<typeof requestZ, string, Thing> {
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
        schema: requestZ,
        fetch: async (req) => await fetchRequest(req),
        matches: (r, req) => req.minSize == null || r.size >= req.minSize,
      },
    });
    this.store = store;
  }

  readonly store: query.Table<string, Thing>;
}

const expectDeleted = <D extends query.Data>(
  value: query.Cached<D> | undefined,
): Deleted<D> => {
  if (!Deleted.matches<D>(value)) throw new Error("expected a deleted answer");
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

    it("resolves a bare key as { key } shorthand through the same entry", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      const result = await client.retrieve("a");
      expect(result).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      expect(client.getCached({ key: "a" })).toEqual(thing("a"));
      expect(fetchKeys).toHaveBeenCalledTimes(1);
    });

    it("resolves a bare key array as { keys } shorthand", async () => {
      const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
      const client = new Client(newCache(), fetchKeys, async () => []);
      const result = await client.retrieve(["a", "b"]);
      expect(result).toEqual([thing("a"), thing("b")]);
      expect(fetchKeys).toHaveBeenCalledWith(["a", "b"]);
    });

    it("rejects a bare key array when the request schema lacks keys", async () => {
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
      await expect(client.retrieve(["a"])).rejects.toThrow(ValidationError);
    });

    it("serves a cached row without a network fetch", async () => {
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

    it("delivers deletion through onChange and flips retrieve to not-found", async () => {
      const client = new Client(
        newCache(),
        async (keys) => keys.map((k) => thing(k)),
        async () => [],
      );
      await client.retrieve({ key: "a" });
      const handler = vi.fn();
      const stop = client.onChange({ key: "a" }, handler);
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
      expect(client.getCached({ key: "a" })).toEqual(thing("a"));
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

    it("treats a request with keys plus another set field as non-keys-only", async () => {
      const fetchRequest = vi.fn(async () => [thing("a", 9)]);
      const client = new Client(newCache(), async () => [], fetchRequest);
      await client.retrieve({ keys: ["a"], minSize: 5 });
      expect(fetchRequest).toHaveBeenCalledTimes(1);
    });

    it("maintains subscribed request answers via matches", async () => {
      const client = new Client(
        newCache(),
        async () => [],
        async () => [thing("a", 5)],
      );
      const handler = vi.fn();
      const stop = client.onChange({ minSize: 3 }, handler);
      await client.retrieve({ minSize: 3 });
      client.store.set([thing("b", 4)]);
      expect(handler).toHaveBeenLastCalledWith([thing("a", 5), thing("b", 4)]);
      stop();
    });

    it("hashes equivalent requests identically after schema normalization", async () => {
      const fetchRequest = vi.fn(async () => [thing("a", 5)]);
      const client = new Client(newCache(), async () => [], fetchRequest);
      const handler = vi.fn();
      const stop = client.onChange({ minSize: 3 }, handler);
      await client.retrieve({ minSize: 3 });
      await client.retrieve({ minSize: 3, searchTerm: undefined });
      expect(fetchRequest).toHaveBeenCalledTimes(1);
      stop();
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
        typeof requestZ,
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
              schema: requestZ,
              fetch: async (req) => await fetchRequest(req),
            },
            single: {
              schema: z.strictObject({ key: z.string() }).transform(({ key }) => key),
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
        await expect(
          client.retrieve({ key: "a", detailsSchema: undefined } as never),
        ).rejects.toThrow(ValidationError);
        expect(fetchRequest).not.toHaveBeenCalled();
      });

      it("routes a clean { key } param through the single space", async () => {
        const fetchKeys = vi.fn(async (keys: string[]) => keys.map((k) => thing(k)));
        const client = new StrictSingleClient(newCache(), fetchKeys, async () => []);
        const result = await client.retrieve({ key: "a" });
        expect(result).toEqual(thing("a"));
        expect(fetchKeys).toHaveBeenCalledWith(["a"]);
      });

      it("retries a bare key as { key } against the single schema", async () => {
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
        await expect(client.retrieve("a")).rejects.toThrow(ValidationError);
        expect(fetchRequest).not.toHaveBeenCalled();
      });
    });
  });
});
