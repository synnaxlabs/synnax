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
import { isConnectionError, NotFoundError, UnexpectedError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import {
  align,
  countBefore,
  RAW_LEVEL,
  type RawBlock,
  rawKey,
  split,
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

// A request to fill a read's raw block up to at least end.
interface Request {
  block: RawBlock;
  end: TimeStamp;
}

// One Core read from start through the end of the blocks first to last, shared by every
// channel whose blocks need that span.
interface Group {
  first: number;
  last: number;
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
   * Deadline for each Core call of a fetch. A call that does not settle within it
   * rejects the reads it serves with an Unreachable error, and its late answer changes
   * no block.
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
 * Reads raw telemetry through raw blocks of the store. A read fetches the blocks it
 * needs from the end of their coverage, with the channel's index, in one Core read for
 * each run of adjacent blocks. Fetched series split at block edges and get block
 * alignment. A fetch failure rejects only the reads that wait on it.
 */
export class Reader {
  private readonly props: Required<Omit<ReaderProps, "batchDebounce">>;
  private readonly batcher: debounce.Batcher<Request>;
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
   * Reads the given time range for the given channel. Series are not cut to the range,
   * so the result can hold samples outside it. Series come in time order. A streamed
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
      waits.push(b.pending ?? this.enqueue(b, TimeStamp.min(tr.end, b.range.end)));
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
    // A streamed series stays in the block it starts in, so one that crosses into the
    // read's first block is in the block before it.
    const before = this.props.store.findRaw(ch, blocks[0].index - 1);
    const series = assemble(tr, before == null ? blocks : [before, ...blocks]);
    // A fetch with a new data type drops the channel's blocks from the store, but this
    // read took its blocks before the fetch, so they can hold the old data type.
    if (new Set(series.map((s) => s.dataType.toString())).size > 1)
      return await this.read(tr, ch);
    return new MultiSeries(series);
  }

  private enqueue(block: RawBlock, end: TimeStamp): Promise<void> {
    return this.props.store.hold(block, this.batcher.enqueue({ block, end }));
  }

  private async batchRead(entries: Array<debounce.Entry<Request>>): Promise<void> {
    const byChannel = new Map<channel.Key, Array<debounce.Entry<Request>>>();
    for (const entry of entries) {
      const { block, end } = entry.req;
      // A fetch that landed while the request waited may have covered it.
      if (!block.coveredEnd.before(end)) {
        entry.resolve();
        continue;
      }
      const existing = byChannel.get(block.channel);
      if (existing == null) byChannel.set(block.channel, [entry]);
      else existing.push(entry);
    }
    const groups = new Map<string, Group>();
    byChannel.forEach((pending) =>
      runs(pending).forEach((run) => {
        const key = `${run.first}/${run.last}/${run.start.valueOf()}`;
        const group = groups.get(key);
        if (group == null) groups.set(key, run);
        else group.entries.push(...run.entries);
      }),
    );
    await Promise.all([...groups.values()].map(async (g) => await this.fetch(g)));
  }

  // Never throws: every outcome settles the group's entries.
  private async fetch(group: Group): Promise<void> {
    const { entries } = group;
    const tr = new TimeRange(
      group.start,
      tileRange({ level: RAW_LEVEL, index: group.last }).end,
    );
    try {
      const refetch = await this.load(group, tr);
      entries.forEach(({ req, resolve }) => {
        if (!refetch.has(req)) return resolve();
        resolve(this.enqueue(req.block, req.end));
      });
    } catch (err) {
      const byChannel = Map.groupBy(entries, ({ req }) => req.block.channel);
      // One channel's error fails a read of several, so each channel reads again alone
      // and gets its own result.
      if (byChannel.size > 1 && !isConnectionError(err))
        await Promise.all(
          [...byChannel.values()].map(
            async (channelEntries) =>
              await this.fetch({ ...group, entries: channelEntries }),
          ),
        );
      else entries.forEach(({ reject }) => reject(err));
    }
  }

  // Reads the requested channels and their indexes over tr, splits the result at block
  // edges, and fills each block whose coverage still ends where the read starts in it.
  // A new data type resets a block, so a block that the read started after its start
  // comes back in the result, to fetch again.
  private async load(
    { first, last, start, entries }: Group,
    tr: TimeRange,
  ): Promise<Set<Request>> {
    const { readRemote, retrieveChannels, store, now } = this.props;
    const committed = now().sub(COMMIT_MARGIN);
    const requested = new Map<channel.Key, Request[]>();
    entries.forEach(({ req }) => {
      const existing = requested.get(req.block.channel);
      if (existing == null) requested.set(req.block.channel, [req]);
      else existing.push(req);
    });
    const channels = [...requested.keys()];
    const refetch = new Set<Request>();
    const payloads = await this.timed(
      retrieveChannels(channels),
      `retrieve of channels ${channels.join(", ")}`,
    );
    if (this.closed) return refetch;
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
    const frame = await this.timed(
      readRemote(tr, [...indexOf.keys()]),
      `block read for ${tr.toString()}`,
    );
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
      const timestamps = idx === 0 ? [] : (grouped.get(idx) ?? []);
      const parts = split(first, last, timestamps, series);
      const targets =
        requested.get(key) ??
        parts.map((_, i) => ({ block: store.raw(key, first + i), end: tr.end }));
      targets.forEach((r) => {
        const { block } = r;
        const from = block.index === first ? start : block.range.start;
        if (changed) {
          block.reset();
          if (!from.equals(block.range.start)) {
            refetch.add(r);
            return;
          }
        }
        if (!block.coveredEnd.equals(from)) return;
        const part = parts[block.index - first];
        const coveredEnd = TimeStamp.max(
          from,
          TimeStamp.min(block.range.end, committed),
        );
        const base = block.coveredCount;
        block.fill(
          align(block.index, base, part.timestamps, part.series),
          coveredEnd,
          base + countBefore(part.timestamps, coveredEnd),
        );
        store.set(rawKey(key, block.index), block);
      });
    });
    return refetch;
  }

  // Rejects with Unreachable once the call outlasts the fetch timeout. A late answer
  // never resumes the fetch, so it changes no block.
  private async timed<T>(call: Promise<T>, name: string): Promise<T> {
    const { fetchTimeout } = this.props;
    return await sync.withTimeout(
      call,
      fetchTimeout,
      () =>
        new Unreachable({
          message: `${name} timed out after ${fetchTimeout.toString()}`,
        }),
    );
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

// Splits one channel's pending requests into runs of adjacent blocks that one Core read
// can fill: a run starts at the coverage end of its first block, and every later block
// in it has no coverage.
const runs = (entries: Array<debounce.Entry<Request>>): Group[] => {
  const sorted = [...entries].sort((a, b) => a.req.block.index - b.req.block.index);
  const out: Group[] = [];
  for (const entry of sorted) {
    const { index, coveredEnd, range } = entry.req.block;
    const run = out.at(-1);
    const joins =
      run != null &&
      index <= run.last + 1 &&
      coveredEnd.equals(index === run.first ? run.start : range.start);
    if (joins) {
      run.last = Math.max(run.last, index);
      run.entries.push(entry);
    } else out.push({ first: index, last: index, start: coveredEnd, entries: [entry] });
  }
  return out;
};

// Returns the series of the blocks that overlap tr in time order. The fetched form of a
// span wins: a streamed series that overlaps a fetched one is left out.
const assemble = (tr: TimeRange, blocks: RawBlock[]): Series[] => {
  const fetched = blocks.flatMap((b) =>
    b.fetched
      .filter((s) => s.timeRange.overlapsWith(tr))
      .map((series) => ({ series, range: series.timeRange })),
  );
  const all = blocks.flatMap((b) => b.fetched);
  const streamed = blocks
    .flatMap((b) => b.streamedIn(tr))
    .filter(({ range }) => !all.some((f) => f.timeRange.overlapsWith(range)));
  return [...fetched, ...streamed]
    .sort((a, b) => TimeRange.sort(a.range, b.range))
    .map(({ series }) => series);
};
