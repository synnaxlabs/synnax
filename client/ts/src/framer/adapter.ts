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
import { NotFoundError, ValidationError } from "@/errors";
import { Codec } from "@/framer/codec";
import { type CrudeFrame, Frame } from "@/framer/frame";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

/** Fetches channel payloads. Missing channels are omitted, not thrown. */
export type ChannelRetriever = (channels: channel.Params) => Promise<channel.Payload[]>;

/** Fetches channel payloads, throwing a {@link NotFoundError} when any is missing. */
const retrieveRequired = async (
  retrieve: ChannelRetriever,
  channels: ReturnType<typeof channel.analyzeParams>["normalized"],
): Promise<channel.Payload[]> => {
  const results = await retrieve(channels);
  const found = new Set<channel.Key | channel.Name>();
  results.forEach(({ key, name }) => found.add(key).add(name));
  const notFound = channels.filter((v) => !found.has(v));
  if (notFound.length > 0)
    throw new NotFoundError(`channels not found matching ${JSON.stringify(notFound)}`);
  return results;
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
    const { variant, normalized } = channel.analyzeParams(channels);
    const fetched = await retrieveRequired(this.retrieveChannels, normalized);
    const newKeys = fetched.map((c) => c.key);
    const newKeySet = new Set(newKeys);
    if (newKeySet.size === this.keys.size && newKeySet.isSubsetOf(this.keys))
      return false;
    this.fetched = fetched;
    if (variant === "keys") {
      this.adapter = null;
      this.keys = new Set(normalized);
      return true;
    }
    this.adapter = new Map(fetched.map((c) => [c.key, c.name]));
    this.keys = new Set(this.adapter.keys());
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
      if (typeof col === "number") {
        const name = adapter.get(col);
        if (name == null) return [col, arr, false];
        return [name, arr, true];
      }
      return [col, arr, true];
    });
  }
}

export class WriteAdapter {
  private byName: Map<channel.Name, channel.Payload> | null;
  private byKey: Map<channel.Key, channel.Payload>;
  private readonly retrieveChannels: ChannelRetriever;
  keys: channel.Key[];
  codec: Codec;

  private constructor(retrieveChannels: ChannelRetriever) {
    this.retrieveChannels = retrieveChannels;
    this.byName = null;
    this.byKey = new Map();
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
    const { normalized } = channel.analyzeParams(channels);
    const results = await retrieveRequired(this.retrieveChannels, normalized);
    const newKeys = results.map((c) => c.key);
    const previousKeySet = new Set(this.keys);
    const newKeySet = new Set(newKeys);
    if (newKeySet.size === previousKeySet.size && newKeySet.isSubsetOf(previousKeySet))
      return false;
    this.byName = new Map(results.map((c) => [c.name, c]));
    this.byKey = new Map(results.map((c) => [c.key, c]));
    this.keys = newKeys;
    this.codec.update(
      this.keys,
      results.map((c) => c.dataType),
    );
    return true;
  }

  private async fetchChannel(
    ch: channel.Key | channel.Name | channel.Payload,
  ): Promise<channel.Payload> {
    // The channel set is fixed between updates, so cached payloads cannot go stale.
    let cached: channel.Payload | undefined;
    if (typeof ch === "number") cached = this.byKey.get(ch);
    else if (typeof ch === "string") cached = this.byName?.get(ch);
    else cached = this.byKey.get(ch.key);
    if (cached != null) return cached;
    const res = await this.retrieveChannels(ch);
    checkForMultipleOrNoResults("channel", channel.singleWhere(ch), res, true);
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
      if (this.byName == null) return fr;
      const cols = fr.columns.map((col_) => {
        const col = typeof col_ === "string" ? this.byName?.get(col_)?.key : col_;
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
      const pld = await this.fetchChannel(k);
      const s = new Series({ data: v, dataType: pld.dataType });
      cols.push(pld.key);
      data.push(s);
    }

    return new Frame(cols, data);
  }
}
