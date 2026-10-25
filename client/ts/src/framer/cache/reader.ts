// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import {
  debounce,
  errors,
  MultiSeries,
  type Reduction,
  type Series,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { NotFoundError, UnexpectedError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import {
  align,
  countBefore,
  RAW_LEVEL,
  type RawBlock,
  rawKey,
} from "@/framer/cache/block";
import { indexes, tileRange } from "@/framer/cache/grid";
import { type Store } from "@/framer/cache/store";
import { type Frame } from "@/framer/frame";

/** Reads a telemetry frame from the Core, reduced when a reduction is given. */
export interface RemoteReader {
  (tr: TimeRange, keys: channel.Key[], reduction?: Reduction): Promise<Frame>;
}

export const DEFAULT_BATCH_DEBOUNCE = TimeSpan.milliseconds(50);

// Samples stamped this close to a fetch may commit after it answers, so a fetch never
// marks them as covered.
const COMMIT_MARGIN = TimeSpan.seconds(1);

// The most block fetches in flight at once.
const MAX_FETCHES = 4;

// A request to fill a read's raw block up to at least end.
interface Request {
  block: RawBlock;
  end: TimeStamp;
}

// Block fetches that start at the same time in the same block share one Core read.
interface Group {
  index: number;
  start: TimeStamp;
  entries: Array<debounce.Entry<Request>>;
}

export interface ReaderProps {
  /** Function used to read remote data from the server. */
  readRemote: RemoteReader;
  /** Looks up the index of each channel a fetch reads. */
  retrieveChannels: ChannelRetriever;
  /** Caches fetched blocks, and holds the blocks the stream writes into. */
  store: Store;
  /**
   * Used to batch read requests to the server to minimize traffic. The window is not
   * extended by later reads, so it is also the maximum wait before a batch fires.
   * @default TimeSpan.milliseconds(50)
   */
  batchDebounce?: TimeSpan;
  /**
   * Deadline for a single remote fetch. A fetch that does not settle within it rejects
   * the reads it serves with an Unreachable error.
   * @default TimeSpan.seconds(30)
   */
  fetchTimeout?: TimeSpan;
  /**
   * The current time on the Core's clock, which stamps every sample.
   * @default TimeStamp.now
   */
  now?: () => TimeStamp;
}

/**
 * Reads raw telemetry through raw blocks of the store. A read fetches each block it
 * needs from the end of the block's coverage to the block's end, with the channel's
 * index, and gives fetched series block alignment. A fetch failure rejects only the
 * reads that wait on it.
 */
export class Reader {
  private readonly props: Required<Omit<ReaderProps, "batchDebounce">>;
  private readonly batcher: debounce.Batcher<Request>;
  private active = 0;
  // Fetches waiting for a slot, in arrival order. Each is handed a freed slot.
  private readonly waiting: Array<() => void> = [];
  private closed = false;

  constructor(props: ReaderProps) {
    const {
      readRemote,
      retrieveChannels,
      store,
      batchDebounce = DEFAULT_BATCH_DEBOUNCE,
      fetchTimeout = TimeSpan.seconds(30),
      now = () => TimeStamp.now(),
    } = props;
    this.props = { readRemote, retrieveChannels, store, fetchTimeout, now };
    this.batcher = new debounce.Batcher({
      interval: batchDebounce,
      exec: async (entries) => await this.batchRead(entries),
    });
  }

  /**
   * Reads the given time range for the given channel. Each series is whole, so the
   * result can hold samples outside the range. Series come in time order. A streamed
   * series that overlaps a fetched one is left out.
   * @throws {UnexpectedError} if the reader is closed while the read is pending.
   * @throws {Unreachable} if a fetch serving the read exceeds the fetch timeout.
   * @throws {NotFoundError} if the channel does not exist.
   */
  async read(tr: TimeRange, ch: channel.Key): Promise<MultiSeries> {
    if (this.closed) throw new UnexpectedError("telemetry reader is closed");
    const blocks = indexes(tr, RAW_LEVEL).map((i) => this.props.store.raw(ch, i));
    const waits: Array<Promise<void>> = [];
    for (const b of blocks) {
      if (b.covers(tr)) continue;
      b.pending ??= this.enqueue(b, TimeStamp.min(tr.end, b.range.end));
      waits.push(b.pending);
    }
    // The read settles after its last fetch, and its first failure wins.
    let failure: Error | undefined;
    await Promise.all(
      waits.map(async (w) => {
        try {
          await w;
        } catch (e) {
          failure ??= errors.fromUnknown(e);
        }
      }),
    );
    if (failure != null) throw failure;
    return assemble(tr, blocks);
  }

  private enqueue(block: RawBlock, end: TimeStamp): Promise<void> {
    const pending = this.batcher.enqueue({ block, end }).finally(() => {
      if (block.pending === pending) block.pending = null;
    });
    return pending;
  }

  private async batchRead(entries: Array<debounce.Entry<Request>>): Promise<void> {
    const groups = new Map<string, Group>();
    for (const entry of entries) {
      const { block, end } = entry.req;
      const { index, coveredEnd } = block;
      // A fetch that landed while the request waited may have covered it.
      if (!coveredEnd.before(end)) {
        entry.resolve();
        continue;
      }
      const key = `${index}/${coveredEnd.valueOf()}`;
      let group = groups.get(key);
      if (group == null) {
        group = { index, start: coveredEnd, entries: [] };
        groups.set(key, group);
      }
      group.entries.push(entry);
    }
    await Promise.all([...groups.values()].map(async (g) => await this.fetch(g)));
  }

  // Never throws: every outcome settles the group's entries.
  private async fetch({ index, start, entries }: Group): Promise<void> {
    const { fetchTimeout } = this.props;
    await this.acquire();
    try {
      const tr = new TimeRange(start, tileRange({ level: RAW_LEVEL, index }).end);
      const requests = entries.map(({ req }) => req);
      const refetch = await sync.withTimeout(
        this.load(index, tr, requests),
        fetchTimeout,
        () =>
          new Unreachable({
            message: `block read for ${tr.toString()} timed out after ${fetchTimeout.toString()}`,
          }),
      );
      entries.forEach(({ req, resolve }) => {
        if (!refetch.has(req)) return resolve();
        req.block.pending = this.enqueue(req.block, req.end);
        resolve(req.block.pending);
      });
    } catch (err) {
      entries.forEach(({ reject }) => reject(err));
    } finally {
      this.release();
    }
  }

  // Reads the requested channels and their indexes over tr, and fills each block whose
  // coverage still ends at tr's start. A new data type resets a block, so a block that
  // the fetch started after its start comes back in the result, to fetch again.
  private async load(
    index: number,
    tr: TimeRange,
    requests: Request[],
  ): Promise<Set<Request>> {
    const { readRemote, retrieveChannels, store, now } = this.props;
    const coveredEnd = TimeStamp.max(
      tr.start,
      TimeStamp.min(tr.end, now().sub(COMMIT_MARGIN)),
    );
    const requested = new Map<channel.Key, Request[]>();
    requests.forEach((r) => {
      const existing = requested.get(r.block.channel);
      if (existing == null) requested.set(r.block.channel, [r]);
      else existing.push(r);
    });
    const channels = [...requested.keys()];
    const payloads = await retrieveChannels(channels);
    // indexOf holds the index of each channel read, or 0 for a channel with none.
    const indexOf = new Map<channel.Key, channel.Key>();
    payloads.forEach((p) => {
      const idx = p.isIndex ? p.key : p.index;
      indexOf.set(p.key, idx);
      if (idx !== 0) indexOf.set(idx, idx);
    });
    channels.forEach((ch) => {
      if (!indexOf.has(ch)) throw new NotFoundError(`channel ${ch} not found`);
    });
    const frame = await readRemote(tr, [...indexOf.keys()]);
    const refetch = new Set<Request>();
    if (this.closed) return refetch;
    const grouped = new Map<channel.Key, Series[]>();
    frame.forEach((k, s) => {
      const key = k as channel.Key;
      const existing = grouped.get(key);
      if (existing == null) grouped.set(key, [s]);
      else existing.push(s);
    });
    indexOf.forEach((idx, key) => {
      const series = grouped.get(key) ?? [];
      const changed =
        series.length > 0 && store.retainDataType(key, series[0].dataType);
      const targets = requested.get(key) ?? [
        { block: store.raw(key, index), end: tr.end },
      ];
      const timestamps = idx === 0 ? [] : (grouped.get(idx) ?? []);
      targets.forEach((r) => {
        const { block } = r;
        if (changed) {
          block.reset();
          if (!tr.start.equals(block.range.start)) {
            refetch.add(r);
            return;
          }
        }
        if (!block.coveredEnd.equals(tr.start)) return;
        const base = block.coveredCount;
        block.fill(
          align(index, base, timestamps, series),
          coveredEnd,
          base + countBefore(timestamps, coveredEnd),
        );
        store.set(rawKey(key, index), block);
      });
    });
    return refetch;
  }

  private async acquire(): Promise<void> {
    if (this.active < MAX_FETCHES) {
      this.active++;
      return;
    }
    await new Promise<void>((resolve) => this.waiting.push(resolve));
  }

  private release(): void {
    const next = this.waiting.shift();
    if (next == null) this.active--;
    else next();
  }

  /**
   * Closes the reader, rejecting every pending read with an UnexpectedError. In-flight
   * fetch results are discarded when they arrive.
   */
  async close(): Promise<void> {
    this.closed = true;
    this.batcher.close(new UnexpectedError("telemetry reader is closed"));
  }
}

// Returns the series of the blocks that overlap tr in time order. The fetched form of a
// span wins: a streamed series that overlaps a fetched one is left out.
const assemble = (tr: TimeRange, blocks: RawBlock[]): MultiSeries => {
  const fetched = blocks.flatMap((b) =>
    b.fetched
      .filter((s) => s.timeRange.overlapsWith(tr))
      .map((series) => ({ series, range: series.timeRange })),
  );
  const all = blocks.flatMap((b) => b.fetched);
  const streamed = blocks
    .flatMap((b) => b.streamedIn(tr))
    .filter(({ range }) => !all.some((f) => f.timeRange.overlapsWith(range)));
  return new MultiSeries(
    [...fetched, ...streamed]
      .sort((a, b) => TimeRange.sort(a.range, b.range))
      .map(({ series }) => series),
  );
};
