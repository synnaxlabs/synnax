// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeSeries, DataType, Series } from "@synnaxlabs/x";

import { channel } from "@/channel";
import { NotFoundError, ValidationError } from "@/errors";
import { Codec } from "@/framer/codec";
import { type CrudeFrame, Frame } from "@/framer/frame";
import { util } from "@/util";

/** Fetches channel payloads. Missing channels are omitted, not thrown. */
export type ChannelRetriever = (channels: channel.Params) => Promise<channel.Payload[]>;

/** Fetches channel payloads, throwing a {@link NotFoundError} when any is missing. */
export const retrieveRequired = async (
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
    const newKeys = new Set(fetched.map((c) => c.key));
    this.adapter =
      variant === "keys" ? null : new Map(fetched.map((c) => [c.key, c.name]));
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

  /**
   * Keeps the series of the requested channels, keyed by name when the channels were
   * requested by name, and adds the time series of each clock that a kept series names.
   */
  adapt(frm: Frame): Frame {
    const hidden = this.hidden(frm);
    const out = this.keep(frm, hidden);
    let named: number[] | undefined;
    for (const s of out.series)
      if (s.clock !== 0 && !(named ??= []).includes(s.clock)) named.push(s.clock);
    if (named == null) return out;
    const clocks = named;
    frm.forEach((k, s) => {
      if (s.clock === k && clocks.includes(s.clock)) out.pushTime(s);
    });
    return out;
  }

  private keep(frm: Frame, hidden: Set<channel.Key> | undefined): Frame {
    const kept = (k: channel.Key, s: Series): boolean =>
      this.keys.has(k) && !(s.clock === k && hidden?.has(k) === true);
    if (this.adapter == null) {
      let shouldFilter = false;
      frm.forEach((k, s) => {
        if (!kept(k as channel.Key, s)) shouldFilter = true;
      });
      if (shouldFilter) return frm.filter((k, s) => kept(k as channel.Key, s));
      return frm;
    }
    const { adapter } = this;
    return frm.mapFilter((col, arr) => {
      if (typeof col === "number") {
        const name = adapter.get(col);
        if (name == null || !kept(col, arr)) return [col, arr, false];
        return [name, arr, true];
      }
      return [col, arr, true];
    });
  }

  // Returns the requested index channels whose stamps in frm time only a calculation,
  // as a calculation's frame holds its primary's stamps under that index's key. Those
  // stamps are time, not samples of the index channel.
  private hidden(frm: Frame): Set<channel.Key> | undefined {
    if (!this.fetched.some((ch) => ch.expression !== "")) return undefined;
    const calculated = new Set<number>();
    const recorded = new Set<number>();
    frm.forEach((k, s) => {
      const record = this.fetched.find((ch) => ch.key === k);
      if (record == null || s.clock === k) return;
      if (record.expression !== "") calculated.add(s.clock);
      recorded.add(record.index);
    });
    const hidden = new Set([...calculated].filter((c) => !recorded.has(c)));
    return hidden.size > 0 ? hidden : undefined;
  }
}

const TIME_AND_NO_CHANNELS =
  'The write holds "time" and no channels. An empty series, time or data, makes ' +
  "nothing present";

export class WriteAdapter {
  /**
   * The index channel of each group of the writer's channels, from the writer's open
   * response. A write's "time" goes to the clock of each group that has a channel in
   * the write.
   */
  clocks: channel.Key[] = [];
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
    const where = channel.singleWhere(typeof ch === "object" ? ch.key : ch);
    util.checkForMultipleOrNoResults("channel", { where }, res, true);
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
      let fr = new Frame(columnsOrData);
      let time: Series | undefined;
      if (this.isTimeKey("time") && fr.has("time")) {
        time = fr.get("time").series[0];
        fr = fr.filter((k) => k !== "time");
      }
      if (this.byName == null) return time == null ? fr : this.placeTime(fr, time);
      const cols = fr.columns.map((col_) => {
        const col = typeof col_ === "string" ? this.byName?.get(col_)?.key : col_;
        if (col == null)
          throw new ValidationError(`
          Channel ${col_} was not provided in the list of channels when opening the writer
        `);
        return col;
      });
      const adapted = new Frame(cols, fr.series);
      return time == null ? adapted : this.placeTime(adapted, time);
    }

    const cols = [];
    const data = [];
    let time: CrudeSeries | undefined;
    const kvs = Object.entries(columnsOrData);
    for (let i = 0; i < kvs.length; i++) {
      const [k, v] = kvs[i];
      if (this.isTimeKey(k)) {
        time = v;
        continue;
      }
      const pld = await this.fetchChannel(k);
      const s = new Series({ data: v, dataType: pld.dataType });
      cols.push(pld.key);
      data.push(s);
    }

    const frame = new Frame(cols, data);
    return time == null ? frame : this.placeTime(frame, time);
  }

  // isTimeKey reports whether k is the reserved "time" key of a write, which a writer
  // channel named time takes instead.
  private isTimeKey(k: string): boolean {
    return k === "time" && this.byName?.has(k) !== true;
  }

  // placeTime returns frame with time under the clock of each group that has a channel
  // in it, and with the clock of each series on it set.
  private placeTime(frame: Frame, time: CrudeSeries): Frame {
    if (frame.columns.length === 0) throw new ValidationError(TIME_AND_NO_CHANNELS);
    const keys = frame.columns as channel.Key[];
    const placed: channel.Key[] = [];
    const series = frame.series.map((s, i) => {
      const ch = this.byKey.get(keys[i]);
      if (ch == null || ch.virtual || ch.index === 0) return s;
      if (!this.clocks.includes(ch.index))
        throw new ValidationError(
          `${ch.name} takes its time from the stamps that its index already stores, ` +
            'so the write cannot hold "time" for it',
        );
      if (keys.includes(ch.index))
        throw new ValidationError(
          `The write holds "time" and the index channel ${ch.index}, which both give ` +
            "the time of one group",
        );
      if (!placed.includes(ch.index)) placed.push(ch.index);
      return new Series({
        data: s.data,
        dataType: s.dataType,
        timeRange: s.timeRange,
        alignment: s.alignment,
        clock: ch.index,
      });
    });
    if (placed.length === 0)
      throw new ValidationError('The write holds "time" and no channel that stores it');
    const data = time instanceof Series ? time.data : time;
    const stamps = placed.map(
      (index) => new Series({ data, dataType: DataType.TIMESTAMP, clock: index }),
    );
    return new Frame([...keys, ...placed], [...series, ...stamps]);
  }
}
