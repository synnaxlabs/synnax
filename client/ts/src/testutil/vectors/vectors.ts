// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";

import { caseconv, deep, id, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterAll, beforeAll, expect, it } from "vitest";
import type z from "zod";

import type Synnax from "@/client";
import type { label } from "@/label";
import { spyOnSend, waitForStreamLive } from "@/testutil";

// The longest each wait for quiet lists runs. Lists stay busy while other spec files
// write to the same Core.
const FETCH_TIMEOUT = TimeSpan.seconds(5);
// Five times the 100 ms debounce of a list's refetch.
const QUIET = TimeSpan.milliseconds(500);
// Covers the connect, the writes, and both waits for quiet lists.
const SETUP_TIMEOUT = TimeSpan.seconds(30);

const VECTORS_URL = new URL(
  "../../../../../core/testdata/filter/vectors.json",
  import.meta.url,
);

/** A filter case: the records that where matches, by alias. */
export interface Case {
  name: string;
  /** The where as the vectors hold it, with camel case fields and vector keys. */
  where: unknown;
  expected: string[];
  serverComputed?: boolean;
}

/** The parsed records, keyed by alias, and the cases of one entity. */
export interface Table<R, W> {
  records: [alias: string, record: R][];
  cases: Case[];
  whereZ: z.ZodType<W>;
}

interface RawVectors {
  records: Record<string, unknown>;
  cases: Case[];
}

/**
 * Reads the filter vectors of the entity, such as "ranges", and parses its records with
 * the given schema. The wheres stay raw until a store maps their keys.
 */
export const read = <R, W>(
  entity: string,
  recordZ: z.ZodType<R>,
  whereZ: z.ZodType<W>,
): Table<R, W> => {
  const all = caseconv.snakeToCamel(
    JSON.parse(readFileSync(VECTORS_URL, "utf-8")),
  ) as Record<string, RawVectors>;
  const { records, cases } = all[entity];
  return {
    records: Object.entries(records).map(([alias, r]) => [alias, recordZ.parse(r)]),
    cases,
    whereZ,
  };
};

/** A live list of records with keys of type K. */
export interface List<K> {
  retrieve: () => Promise<unknown>;
  getCached: () => { records: Array<{ key: K }> } | undefined;
  onChange: (handler: () => void) => () => void;
}

/** The records a store wrote to the Core, and how the vectors name them. */
export interface Written<K> {
  /** The key of each record the store wrote, by alias. */
  keys: Map<string, K>;
  /**
   * Maps each vector value that names a record, such as a key, a foreign key, or a name
   * the Core requires to be unique, to the value the Core holds.
   */
  values: Map<unknown, unknown>;
}

/** A client store whose live lists a vector table checks. */
export interface Store<R, W, K> {
  client: Synnax;
  /** The unary target that the store's lists fetch from. */
  target: string;
  /**
   * Writes the records the client can write before any list opens, such as the
   * dependencies of the records and records whose keys the Core assigns. Returns what
   * the lists need to map the wheres.
   */
  prepare: (records: [alias: string, record: R][]) => Promise<Written<K>>;
  /**
   * Writes the vector records, or brings the prepared ones to their vector values,
   * after every list opened. The lists decide on each change themselves.
   */
  write: (records: [alias: string, record: R][]) => Promise<void>;
  watch: (where: W) => List<K>;
}

const mapValues = (value: unknown, values: Map<unknown, unknown>): unknown => {
  if (values.has(value)) return values.get(value);
  if (Array.isArray(value)) return value.map((v) => mapValues(v, values));
  if (value != null && typeof value === "object")
    return Object.fromEntries(
      Object.entries(value).map(([k, v]) => [k, mapValues(v, values)]),
    );
  return value;
};

interface Opened<K, W> {
  where: W;
  list: List<K>;
  close: () => void;
}

/**
 * Adds a spec for each case. One setup opens a live list per case through the store,
 * then writes the records, so each list admits or drops them itself. A case that the
 * vectors mark as server-computed must fetch from the Core again. Any other case must
 * not, so the client decides it alone. Each spec checks the records of its list against
 * the case, among the records the store wrote.
 */
export const itFiltersLive = <R, W, K>(
  { records, cases, whereZ }: Table<R, W>,
  open: () => Store<R, W, K>,
): void => {
  let written: Written<K>;
  // The wheres of the requests the lists sent to the Core after the write.
  let fetched: unknown[] = [];
  const opened = new Map<string, Opened<K, W>>();
  const fetchedAgain = (where: W): boolean =>
    fetched.some((sent) => deep.equal(sent, where));

  beforeAll(async () => {
    const store = open();
    await store.client.connect();
    await waitForStreamLive(store.client.connection);
    written = await store.prepare(records);
    await Promise.all(
      cases.map(async ({ name, where }) => {
        const parsed = whereZ.parse(mapValues(where, written.values));
        const list = store.watch(parsed);
        const close = list.onChange(() => {});
        await list.retrieve();
        opened.set(name, { where: parsed, list, close });
      }),
    );
    const send = spyOnSend(store.client);
    try {
      const sent = (): unknown[] =>
        send.mock.calls.flatMap(([target, req]) =>
          target === store.target &&
          typeof req === "object" &&
          req != null &&
          "where" in req
            ? [req.where]
            : [],
        );
      // Waits until no list has fetched for a quiet window, longer than the debounce
      // of a list's refetch, or until the deadline.
      const settle = async (done: () => boolean = () => true): Promise<void> => {
        const deadline = TimeStamp.now().add(FETCH_TIMEOUT);
        let count = sent().length;
        let since = TimeStamp.now();
        while (TimeStamp.now().before(deadline)) {
          await sleep.sleep(TimeSpan.milliseconds(20));
          const next = sent().length;
          if (next !== count) [count, since] = [next, TimeStamp.now()];
          else if (done() && TimeStamp.since(since).greaterThan(QUIET)) return;
        }
      };
      // The refetches that the prepared writes started end first, so every fetch the
      // spec counts starts after the write.
      await settle();
      const before = sent().length;
      await store.write(records);
      const owed = (): boolean => {
        fetched = sent().slice(before);
        return cases.some(({ name, serverComputed }) => {
          const where = opened.get(name)?.where;
          return serverComputed === true && (where == null || !fetchedAgain(where));
        });
      };
      // A list that the client cannot decide refetches after a debounce, so the spy
      // stays until the lists are quiet as well as until each owed fetch arrives.
      await settle(() => !owed());
      fetched = sent().slice(before);
    } finally {
      send.mockRestore();
    }
  }, SETUP_TIMEOUT.milliseconds);

  afterAll(() => opened.forEach(({ close }) => close()));

  const held = (list: List<K>): string[] => {
    const keys = new Set(list.getCached()?.records.map(({ key }) => key));
    return [...written.keys]
      .filter(([, key]) => keys.has(key))
      .map(([alias]) => alias)
      .sort();
  };

  cases.forEach(({ name, expected, serverComputed }) =>
    it(`should keep a live list of the case: ${name}`, async () => {
      const { where, list } = opened.get(name) ?? {};
      if (where == null || list == null) throw new Error(`no list for ${name}`);
      const want = expected.filter((alias) => written.keys.has(alias)).sort();
      await expect.poll(() => held(list)).toEqual(want);
      expect(fetchedAgain(where)).toBe(serverComputed === true);
    }),
  );
};

/** Creates a rack for each vector rack key and maps each vector key to its rack. */
export const createRacks = async (
  client: Synnax,
  keys: number[],
): Promise<Map<number, number>> => {
  const unique = [...new Set(keys)];
  const racks = await client.racks.create(
    unique.map((key) => ({ name: `where-rack-${key}-${id.create()}` })),
  );
  return new Map(unique.map((key, i) => [key, racks[i].key]));
};

/** Creates a label for each vector label key and maps each vector key to its label. */
export const createLabels = async (
  client: Synnax,
  keys: string[],
): Promise<Map<string, string>> => {
  const unique = [...new Set(keys)];
  const labels = await client.labels.create(
    unique.map((): label.New => ({
      name: `where-label-${id.create()}`,
      color: { type: "fixed", value: "#E774D0" },
    })),
  );
  return new Map(unique.map((key, i) => [key, labels[i].key]));
};
