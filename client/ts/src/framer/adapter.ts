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
import { checkForMultipleOrNoResults } from "@/util/retrieve";

/** Fetches channel payloads. Missing channels are omitted, not thrown. */
export type ChannelRetriever = (channels: channel.Params) => Promise<channel.Payload[]>;

const uniqueByKey = (channels: channel.Payload[]): channel.Payload[] => [
  ...new Map(channels.map((c) => [c.key, c])).values(),
];

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
    const lookups = channel.toLookups(channels);
    const resolved = await channel.resolveLookups(lookups, this.retrieveChannels);
    const fetched = uniqueByKey(resolved);
    const newKeys = new Set(fetched.map((c) => c.key));
    // A frame holds keys or names, never both, so a list that holds any key labels by
    // key. Otherwise each channel takes the name or path that looked it up.
    const labels = new Map<channel.Key, string>();
    if (lookups.every((l) => typeof l === "string"))
      lookups.forEach((l, i) => labels.set(resolved[i].key, l));
    this.adapter = labels.size === 0 ? null : labels;
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
  /** The channel of each key of a record frame, resolved once until an update. */
  private byRecordKey: Map<string, channel.Payload>;
  private readonly retrieveChannels: ChannelRetriever;
  keys: channel.Key[];
  codec: Codec;

  private constructor(retrieveChannels: ChannelRetriever) {
    this.retrieveChannels = retrieveChannels;
    this.byText = new Map();
    this.byKey = new Map();
    this.byRecordKey = new Map();
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

  async adaptParams(data: channel.Params): Promise<channel.Key[]> {
    const arrParams = channel.paramsZ.parse(data);
    const keys = await Promise.all(
      arrParams.map(async (p) => await this.adaptToKey(p)),
    );
    return keys;
  }

  async update(channels: channel.Params): Promise<boolean> {
    const lookups = channel.toLookups(channels);
    const resolved = await channel.resolveLookups(lookups, this.retrieveChannels);
    const results = uniqueByKey(resolved);
    this.byText = new Map();
    for (const c of results) this.hold(c.name, c);
    lookups.forEach((l, i) => {
      if (typeof l === "string") this.hold(l, resolved[i]);
    });
    this.byKey = new Map(results.map((c) => [c.key, c]));
    this.byRecordKey = new Map();
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
   * the writer.
   */
  async adaptRecordKeys(keys: string[]): Promise<channel.Key[]> {
    return await Promise.all(
      keys.map(async (k) => await this.adaptToKey(this.recordKey(k))),
    );
  }

  /**
   * @returns the channel that a key of a record frame names. A record keys its columns
   * by string, so a key of all digits is a channel key unless a channel of the writer
   * holds it as a name.
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

  private async fetchChannel(
    ch: channel.Key | channel.Name | channel.Payload,
  ): Promise<channel.Payload> {
    // The channel set is fixed between updates, so cached payloads cannot go stale.
    const cached =
      typeof ch === "string"
        ? this.held(ch)
        : this.byKey.get(typeof ch === "number" ? ch : ch.key);
    if (cached != null) return cached;
    const lookups = channel.toLookups(ch);
    const res = await this.retrieveChannels(lookups);
    checkForMultipleOrNoResults(
      "channel",
      { where: channel.lookupWhere(lookups) },
      res,
      true,
    );
    return res[0];
  }

  private async adaptToKey(k: channel.Key | channel.Name): Promise<channel.Key> {
    if (typeof k === "number") return k;
    const res = await this.fetchChannel(k);
    return res.key;
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

      const pld = await this.fetchChannel(columnsOrData);
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
      const cols = [];
      const data = [];
      for (let i = 0; i < columnsOrData.length; i++) {
        const pld = await this.fetchChannel(columnsOrData[i]);
        if (i >= series.length)
          throw new ValidationError(`
          Received an array of channel names or keys but not enough series.
          `);

        const s = new Series({
          data: series[i] as CrudeSeries,
          dataType: pld.dataType,
        });
        cols.push(pld.key);
        data.push(s);
      }
      return new Frame(cols, data);
    }

    if (columnsOrData instanceof Frame || columnsOrData instanceof Map) {
      const fr = new Frame(columnsOrData);
      const cols = fr.columns.map((col_) => {
        const col = typeof col_ === "string" ? this.held(col_)?.key : col_;
        if (col == null)
          throw new ValidationError(`
          Channel ${col_} was not provided in the list of channels when opening the writer
        `);
        return col;
      });
      return new Frame(cols, fr.series);
    }

    const cols = [];
    const data = [];
    const kvs = Object.entries(columnsOrData);
    for (let i = 0; i < kvs.length; i++) {
      const [k, v] = kvs[i];
      let pld = this.byRecordKey.get(k);
      if (pld == null) {
        pld = await this.fetchChannel(this.recordKey(k));
        this.byRecordKey.set(k, pld);
      }
      const s = new Series({ data: v, dataType: pld.dataType });
      cols.push(pld.key);
      data.push(s);
    }

    return new Frame(cols, data);
  }
}
