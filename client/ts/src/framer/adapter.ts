// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeSeries, Series } from "@synnaxlabs/x";

import { channel } from "@/channel";
import { ValidationError } from "@/errors";
import { Codec } from "@/framer/codec";
import { type CrudeFrame, Frame } from "@/framer/frame";
import { recordColumn } from "@/framer/record";

/**
 * Fetches the channels that a list of lookups names, omitting any that no lookup finds.
 * A request with a name or a path must be strict, so the Core refuses an ambiguous
 * name, and each channel found by a path must hold its path.
 */
export type ChannelRetriever = (channels: channel.Params) => Promise<channel.Payload[]>;

const nfc = (lookup: channel.Lookup): channel.Lookup =>
  typeof lookup === "string" ? lookup.normalize("NFC") : lookup;

/**
 * @returns the channel of each lookup, one per channel, in the order of its first
 * lookup. Lookups compare in NFC, so a repeat of one lookup keeps one channel.
 * @throws {ValidationError} if two different lookups name one channel.
 */
const distinct = (
  lookups: channel.Lookup[],
  each: channel.Payload[],
): channel.Payload[] => {
  if (lookups.length < 2) return each;
  const byKey = new Map<channel.Key, channel.Lookup>();
  const channels: channel.Payload[] = [];
  lookups.forEach((lookup, i) => {
    const ch = each[i];
    const first = byKey.get(ch.key);
    if (first == null) {
      byKey.set(ch.key, lookup);
      channels.push(ch);
    } else if (nfc(first) !== nfc(lookup))
      throw new ValidationError(
        `${JSON.stringify(first)} and ${JSON.stringify(lookup)} name one channel`,
      );
  });
  return channels;
};

/** The channels of an open, a read, or a file read. */
export interface Resolved {
  /** The channels as the caller gave them, each payload as its key. */
  lookups: channel.Lookup[];
  /** The channel of each lookup, at the index of its lookup. */
  each: channel.Payload[];
  /** Each channel once, in the order of its first lookup. */
  channels: channel.Payload[];
}

/**
 * Resolves the channels of an open, a read, or a file read in one retrieve.
 * @throws {NotFoundError} naming every lookup that finds no channel.
 * @throws {ValidationError} if two different lookups name one channel.
 */
export const resolveOpen = async (
  retrieve: ChannelRetriever,
  params: channel.Params,
): Promise<Resolved> => {
  const lookups = channel.toLookups(params);
  const each = await channel.resolveLookups(lookups, retrieve);
  return { lookups, each, channels: distinct(lookups, each) };
};

/**
 * Checks that the entries of one write name different channels.
 * @param texts - the reference of each entry, as the caller gave it.
 * @throws {ValidationError} if two entries name one channel.
 */
const refuseShared = (texts: string[], channels: channel.Payload[]): void => {
  if (channels.length < 2) return;
  const byKey = new Map<channel.Key, string>();
  channels.forEach((ch, i) => {
    const first = byKey.get(ch.key);
    if (first != null)
      throw new ValidationError(
        `${JSON.stringify(first)} and ${JSON.stringify(texts[i])} name one channel`,
      );
    byKey.set(ch.key, texts[i]);
  });
};

export class ReadAdapter {
  private adapter: Map<channel.Key, string> | null;
  private readonly retrieveChannels: ChannelRetriever;
  keys: Set<channel.Key>;
  codec: Codec;
  private fetched: channel.Payload[] = [];

  private constructor(retrieveChannels: ChannelRetriever) {
    this.retrieveChannels = retrieveChannels;
    this.adapter = null;
    this.keys = new Set();
    this.codec = new Codec();
  }

  static async open(
    retrieveChannels: ChannelRetriever,
    channels: channel.Params,
  ): Promise<ReadAdapter> {
    const adapter = new ReadAdapter(retrieveChannels);
    await adapter.update(channels);
    adapter.advance();
    return adapter;
  }

  /**
   * Resolves the channels into the keys of the next request.
   * @returns true when the keys changed.
   */
  async update(channels: channel.Params): Promise<boolean> {
    const {
      lookups,
      each,
      channels: fetched,
    } = await resolveOpen(this.retrieveChannels, channels);
    const newKeys = new Set(fetched.map((c) => c.key));
    // A frame holds keys or names, never both, so a list that holds any key labels by
    // key. Otherwise each channel takes the name or path that first looked it up.
    let labels: Map<channel.Key, string> | null = null;
    if (lookups.every((l) => typeof l === "string")) {
      labels = new Map();
      for (let i = 0; i < lookups.length; i++)
        if (!labels.has(each[i].key)) labels.set(each[i].key, lookups[i]);
    }
    this.adapter = labels;
    if (newKeys.size === this.keys.size && newKeys.isSubsetOf(this.keys)) return false;
    this.fetched = fetched;
    this.keys = newKeys;
    return true;
  }

  /**
   * Advances the codec to the keys, as the Core does for every request that names
   * channels. Call it once before each such request, or frames after it decode as
   * empty.
   */
  advance(): void {
    // The Core skips an update that carries no keys, and both sides must agree on the
    // sequence number of each codec state.
    if (this.fetched.length === 0) return;
    this.codec.update(
      this.fetched.map((c) => c.key),
      this.fetched.map((c) => c.dataType),
    );
  }

  adapt(frm: Frame): Frame {
    if (this.adapter == null) {
      let shouldFilter = false;
      frm.forEach((k) => {
        if (!this.keys.has(k as channel.Key)) shouldFilter = true;
      });
      if (shouldFilter) return frm.filter((k) => this.keys.has(k as channel.Key));
      return frm;
    }
    const { adapter } = this;
    return frm.mapFilter((col, arr) => {
      if (typeof col !== "number") return [col, arr, true];
      const label = adapter.get(col);
      if (label == null) return [col, arr, false];
      return [label, arr, true];
    });
  }
}

export class WriteAdapter {
  /** The held channels by name, and by each path the writer opened with. */
  private byText: Map<string, channel.Payload[]>;
  private byKey: Map<channel.Key, channel.Payload>;
  /**
   * The keys of the last record and their checked channels, until an update. Only a
   * record of held references counts, since a path can move between records.
   */
  private lastRecord: { keys: string[]; channels: channel.Payload[] } | null;
  private readonly retrieveChannels: ChannelRetriever;
  keys: channel.Key[];
  codec: Codec;

  private constructor(retrieveChannels: ChannelRetriever) {
    this.retrieveChannels = retrieveChannels;
    this.byText = new Map();
    this.byKey = new Map();
    this.lastRecord = null;
    this.keys = [];
    this.codec = new Codec();
  }

  static async open(
    retrieveChannels: ChannelRetriever,
    channels: channel.Params,
  ): Promise<WriteAdapter> {
    const adapter = new WriteAdapter(retrieveChannels);
    await adapter.update(channels);
    return adapter;
  }

  /**
   * @returns the key of each channel of data.
   * @throws {NotFoundError} naming every channel that is not found.
   * @throws {ValidationError} if two entries name one channel.
   */
  async adaptParams(data: channel.Params): Promise<channel.Key[]> {
    const refs = channel.paramsZ.parse(data);
    const channels = await this.resolveRefs(refs);
    refuseShared(refs.map(String), channels);
    return channels.map((c) => c.key);
  }

  async update(channels: channel.Params): Promise<boolean> {
    const {
      lookups,
      each,
      channels: results,
    } = await resolveOpen(this.retrieveChannels, channels);
    this.byText = new Map();
    for (const c of results) this.hold(c.name, c);
    for (let i = 0; i < lookups.length; i++) {
      const l = lookups[i];
      if (typeof l === "string") this.hold(l, each[i]);
    }
    this.byKey = new Map(results.map((c) => [c.key, c]));
    this.lastRecord = null;
    const newKeys = results.map((c) => c.key);
    const previousKeySet = new Set(this.keys);
    const newKeySet = new Set(newKeys);
    if (newKeySet.size === previousKeySet.size && newKeySet.isSubsetOf(previousKeySet))
      return false;
    this.keys = newKeys;
    this.codec.update(
      this.keys,
      results.map((c) => c.dataType),
    );
    return true;
  }

  /**
   * @returns the keys of the channels that the keys of a record name, by the same rule
   * as a record frame.
   * @throws {ValidationError} if a key names both a channel key and a channel name of
   * the writer, or if two keys name one channel.
   */
  async adaptRecordKeys(keys: string[]): Promise<channel.Key[]> {
    return (await this.resolveRecord(keys)).map((c) => c.key);
  }

  /**
   * @returns the channel that a key of a record frame names. A key that is a channel
   * key in canonical decimal is a key, unless a channel of the writer holds it as a
   * name. Any other key is a name or a path.
   * @throws {ValidationError} if the writer holds both a channel with that name and a
   * channel with that key.
   */
  private recordKey(k: string): channel.Key | channel.Name {
    const col = recordColumn(k);
    if (typeof col === "string") return col;
    const named = this.held(k);
    if (named == null || named.key === col) return col;
    if (this.byKey.has(col))
      throw new ValidationError(
        `Record column "${k}" names both channel ${col} and the channel named "${k}"`,
      );
    return k;
  }

  /**
   * @returns the channel of each key of a record.
   * @throws {NotFoundError} naming every key that finds no channel.
   * @throws {ValidationError} if two keys name one channel.
   */
  private async resolveRecord(keys: string[]): Promise<channel.Payload[]> {
    const refs = keys.map((k) => this.recordKey(k));
    const channels = this.heldOf(refs);
    const allHeld = channels.every((c) => c != null);
    const resolved = allHeld ? channels : await this.fetchUnheld(refs, channels);
    refuseShared(keys, resolved);
    this.lastRecord = allHeld ? { keys, channels: resolved } : null;
    return resolved;
  }

  /**
   * @returns the channel of each reference.
   * @throws {NotFoundError} naming every reference that finds no channel.
   */
  private async resolveRefs(refs: channel.Lookup[]): Promise<channel.Payload[]> {
    return await this.fetchUnheld(refs, this.heldOf(refs));
  }

  /**
   * @returns the held channel of each reference, by key, NFC name, or the NFC path that
   * opened it, or undefined.
   */
  private heldOf(refs: channel.Lookup[]): (channel.Payload | undefined)[] {
    return refs.map((r) => (typeof r === "number" ? this.byKey.get(r) : this.held(r)));
  }

  /**
   * @returns channels, with each missing one resolved from its reference in one
   * retrieve.
   * @throws {NotFoundError} naming every reference that finds no channel.
   */
  private async fetchUnheld(
    refs: channel.Lookup[],
    channels: (channel.Payload | undefined)[],
  ): Promise<channel.Payload[]> {
    const unheld = refs.filter((_, i) => channels[i] == null);
    if (unheld.length === 0) return channels as channel.Payload[];
    const found = await channel.resolveLookups(unheld, this.retrieveChannels);
    let j = 0;
    return channels.map((ch) => ch ?? found[j++]);
  }

  private hold(text: string, ch: channel.Payload): void {
    const key = text.normalize("NFC");
    const held = this.byText.get(key);
    if (held == null) this.byText.set(key, [ch]);
    else if (!held.some((c) => c.key === ch.key)) held.push(ch);
  }

  /**
   * @returns the held channel that text names, or undefined.
   * @throws {ValidationError} if the writer holds several channels with that name.
   */
  private held(text: string): channel.Payload | undefined {
    const held = this.byText.get(text.normalize("NFC"));
    if (held != null && held.length > 1)
      throw new ValidationError(
        `The writer holds ${held.length} channels named "${text}"`,
      );
    return held?.[0];
  }

  encode(frame: Frame): Uint8Array {
    return this.codec.encode(frame.toPayload());
  }

  async adapt(
    columnsOrData:
      channel.Params | Record<channel.Key | channel.Name, CrudeSeries> | CrudeFrame,
    series?: CrudeSeries | CrudeSeries[],
  ): Promise<Frame> {
    if (typeof columnsOrData === "string" || typeof columnsOrData === "number") {
      if (series == null)
        throw new ValidationError(`
        Received a single channel name or key but no series.
        `);
      if (Array.isArray(series))
        if (series.some((s) => s instanceof Series || Array.isArray(s)))
          throw new ValidationError(`
          Received a single channel name or key but multiple series.
          `);

      const [pld] = await this.resolveRefs([columnsOrData]);
      const s = new Series({ data: series, dataType: pld.dataType });
      return new Frame(pld.key, s);
    }

    if (Array.isArray(columnsOrData)) {
      if (series == null)
        throw new ValidationError(`
          Received an array of channel names or keys but no series.
          `);
      if (!Array.isArray(series))
        throw new ValidationError(`
        Received an array of channel names or keys but no array of series.
        `);
      if (series.length < columnsOrData.length)
        throw new ValidationError(`
          Received an array of channel names or keys but not enough series.
          `);
      const refs = channel.toLookups(columnsOrData);
      const plds = await this.resolveRefs(refs);
      refuseShared(refs.map(String), plds);
      const data = plds.map(
        (pld, i) =>
          new Series({ data: series[i] as CrudeSeries, dataType: pld.dataType }),
      );
      return new Frame(
        plds.map((pld) => pld.key),
        data,
      );
    }

    if (columnsOrData instanceof Frame || columnsOrData instanceof Map) {
      const fr = new Frame(columnsOrData);
      if (fr.colType !== "name") return fr;
      const names = fr.columns as channel.Name[];
      const plds = await this.resolveRefs(names);
      refuseShared(names, plds);
      return new Frame(
        plds.map((pld) => pld.key),
        fr.series,
      );
    }

    const kvs = Object.entries(columnsOrData);
    // A writer often writes records of one shape, which then needs no lookup or check.
    const last = this.lastRecord;
    let plds: channel.Payload[];
    if (
      last != null &&
      last.keys.length === kvs.length &&
      kvs.every(([k], i) => k === last.keys[i])
    )
      plds = last.channels;
    else plds = await this.resolveRecord(kvs.map(([k]) => k));
    const cols = new Array<channel.Key>(kvs.length);
    const data = new Array<Series>(kvs.length);
    for (let i = 0; i < kvs.length; i++) {
      cols[i] = plds[i].key;
      data[i] = new Series({ data: kvs[i][1], dataType: plds[i].dataType });
    }
    return new Frame(cols, data);
  }
}
