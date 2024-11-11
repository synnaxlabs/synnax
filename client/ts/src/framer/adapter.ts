// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeSeries, Series } from "@synnaxlabs/x/telem";

import {
  type Key,
  type KeyOrName,
  type Name,
  type Params,
  type Payload,
} from "@/channel/payload";
import {
  analyzeChannelParams,
  type Retriever,
  retrieveRequired,
} from "@/channel/retriever";
import { ValidationError } from "@/errors";
import { type CrudeFrame, Frame } from "@/framer/frame";

export class ReadFrameAdapter {
  private adapter: Map<Key, Name> | null;
  retriever: Retriever;
  keys: Key[];

  private constructor(retriever: Retriever) {
    this.retriever = retriever;
    this.adapter = null;
    this.keys = [];
  }

  static async open(retriever: Retriever, channels: Params): Promise<ReadFrameAdapter> {
    const adapter = new ReadFrameAdapter(retriever);
    await adapter.update(channels);
    return adapter;
  }

  async update(channels: Params): Promise<void> {
    const { variant, normalized } = analyzeChannelParams(channels);
    if (variant === "keys") {
      this.adapter = null;
      this.keys = normalized as Key[];
      return;
    }
    const fetched = await this.retriever.retrieve(normalized);
    const a = new Map<Key, Name>();
    this.adapter = a;
    normalized.forEach((name) => {
      const channel = fetched.find((channel) => channel.name === name);
      if (channel == null) throw new Error(`Channel ${name} not found`);
      a.set(channel.key, channel.name);
    });
    this.keys = Array.from(this.adapter.keys());
  }

  adapt(columnsOrData: Frame): Frame {
    if (this.adapter == null) return columnsOrData;
    const a = this.adapter;
    return columnsOrData.map((k, arr) => {
      if (typeof k === "number") {
        const name = a.get(k);
        if (name == null) throw new Error(`Channel ${k} not found`);
        return [name, arr];
      }
      return [k, arr];
    });
  }
}

export class WriteFrameAdapter {
  private adapter: Map<Name, Key> | null;
  retriever: Retriever;
  keys: Key[];

  private constructor(retriever: Retriever) {
    this.retriever = retriever;
    this.adapter = null;
    this.keys = [];
  }

  static async open(
    retriever: Retriever,
    channels: Params,
  ): Promise<WriteFrameAdapter> {
    const adapter = new WriteFrameAdapter(retriever);
    await adapter.update(channels);
    return adapter;
  }

  async adaptObjectKeys<V>(data: Record<KeyOrName, V>): Promise<Record<Key, V>> {
    const out: Record<Key, V> = {};
    for (const [k, v] of Object.entries(data)) out[await this.adaptToKey(k)] = v;
    return out;
  }

  async update(channels: Params): Promise<void> {
    const results = await retrieveRequired(this.retriever, channels);
    this.adapter = new Map<Name, Key>(results.map((c) => [c.name, c.key]));
    this.keys = results.map((c) => c.key);
  }

  private async fetchChannel(ch: Key | Name): Promise<Payload> {
    const res = await this.retriever.retrieve(ch);
    if (res.length === 0) throw new Error(`Channel ${ch} not found`);
    return res[0];
  }

  private async adaptToKey(k: KeyOrName): Promise<Key> {
    if (typeof k === "number") return k;
    const res = await this.fetchChannel(k);
    return res.key;
  }

  async adapt(
    columnsOrData: Params | Record<KeyOrName, CrudeSeries> | CrudeFrame,
    series?: CrudeSeries | CrudeSeries[],
  ): Promise<Frame> {
    if (typeof columnsOrData === "string" || typeof columnsOrData === "number") {
      if (series == null)
        throw new ValidationError(`
        Received a single channel name or key but no series.
        `);
      if (Array.isArray(series)) {
        if (series.some((s) => s instanceof Series || Array.isArray(s)))
          throw new ValidationError(`
          Received a single channel name or key but multiple series.
          `);

        series = series as CrudeSeries;
      }
      const pld = await this.fetchChannel(columnsOrData);
      const s = new Series({ data: series as CrudeSeries, dataType: pld.dataType });
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
      if (this.adapter == null) return fr;
      const cols = fr.columns.map((col_) => {
        const col = typeof col_ === "string" ? this.adapter?.get(col_) : col_;
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
