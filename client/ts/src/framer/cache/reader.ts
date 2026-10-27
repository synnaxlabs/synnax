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
  type destructor,
  errors,
  type Reduction,
  scheduler,
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
  type Part,
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

/**
 * Samples stamped this close to a fetch may commit after it answers, so a fetch never
 * marks them as covered.
 */
export const COMMIT_MARGIN = TimeSpan.seconds(1);

// A request to fill a read's raw block up to at least end. A catch-up request trusts
// the read up to the channel's newest sample, since the stream acked the channel before
// it.
interface Request {
  block: RawBlock;
  end: TimeStamp;
  catchUp: boolean;
  // The blocks the read filled, which a catch-up keeps until its caller releases them.
  pinned?: Set<RawBlock>;
}

// One Core read from start to end over the blocks first to last, shared by every
// channel whose blocks need that span. A catch-up read ends at TimeStamp.MAX, and fills
// the blocks after last that hold the channel's newer samples.
interface Group {
  first: number;
  last: number;
  start: TimeStamp;
  end: TimeStamp;
  catchUp: boolean;
  entries: Array<scheduler.BatcherEntry<Request>>;
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
  private readonly batcher: scheduler.Batcher<Request>;
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
    this.batcher = new scheduler.Batcher({
      interval: batchDebounce,
      exec: async (entries) => await this.batchRead(entries),
    });
  }

  /**
   * Fetches the channel's raw blocks over the time range that their samples do not
   * cover yet. Settles once every fetch lands.
   * @throws {UnexpectedError} if the reader is closed while the fill is pending.
   * @throws {Unreachable} if a fetch serving the fill exceeds the fetch timeout.
   * @throws {NotFoundError} if the channel does not exist.
   */
  async fill(tr: TimeRange, ch: channel.Key): Promise<void> {
    if (this.closed) throw new UnexpectedError("telemetry reader is closed");
    const blocks = indexes(tr, RAW_LEVEL).map((i) => this.props.store.raw(ch, i));
    const waits: Array<Promise<void>> = [];
    for (const b of blocks) {
      if (b.covers(tr)) continue;
      waits.push(
        b.pending ??
          this.enqueue({
            block: b,
            end: TimeStamp.min(tr.end, b.range.end),
            catchUp: false,
          }),
      );
    }
    // The fill settles after its last fetch, and its first failure wins.
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
  }

  /**
   * Fetches the channel's samples from the coverage end of raw block first onward into
   * the blocks that hold them, and trusts every sample the read returns. Its cost grows
   * with the blocks that hold samples, not with the span to now. Call it only after the
   * stream acked the channel and its index, so the stream delivers every sample the
   * read misses. The store keeps the blocks the read fills until the returned
   * destructor runs, so the join finds the samples it numbers the stream's samples
   * after.
   * @throws {UnexpectedError} if the reader is closed while the read is pending.
   * @throws {Unreachable} if the fetch exceeds the fetch timeout.
   * @throws {NotFoundError} if the channel does not exist.
   */
  async catchUp(ch: channel.Key, first: number): Promise<destructor.Destructor> {
    if (this.closed) throw new UnexpectedError("telemetry reader is closed");
    const { store } = this.props;
    const block = store.raw(ch, first);
    const pinned = new Set<RawBlock>();
    const release = (): void => store.repin(pinned, new Set());
    try {
      await this.batcher.enqueue({ block, end: TimeStamp.MAX, catchUp: true, pinned });
    } catch (e) {
      release();
      throw errors.fromUnknown(e);
    }
    return release;
  }

  private enqueue(req: Request): Promise<void> {
    return this.props.store.hold(req.block, this.batcher.enqueue(req));
  }

  private async batchRead(
    entries: Array<scheduler.BatcherEntry<Request>>,
  ): Promise<void> {
    const pending = entries.filter((entry) => {
      const { block, end } = entry.req;
      // A fetch that landed while the request waited may have covered it, and a joined
      // block takes its samples from the stream.
      if (!block.joined && block.coveredEnd.before(end)) return true;
      entry.resolve();
      return false;
    });
    const byChannel = Map.groupBy(
      pending,
      ({ req }) => `${req.block.channel}/${req.catchUp}`,
    );
    const groups = new Map<string, Group>();
    byChannel.forEach((pending) =>
      runs(pending).forEach((run) => {
        const key = `${run.first}/${run.last}/${run.start.valueOf()}/${run.catchUp}`;
        const group = groups.get(key);
        if (group == null) groups.set(key, run);
        else {
          group.entries.push(...run.entries);
          group.end = TimeStamp.max(group.end, run.end);
        }
      }),
    );
    await Promise.all([...groups.values()].map(async (g) => await this.fetch(g)));
  }

  // Never throws: every outcome settles the group's entries.
  private async fetch(group: Group): Promise<void> {
    const { entries } = group;
    const tr = new TimeRange(group.start, group.end);
    try {
      const refetch = await this.load(group, tr);
      entries.forEach(({ req, resolve }) => {
        if (!refetch.has(req)) return resolve();
        resolve(this.enqueue(req));
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

  // Reads the channels and their indexes over tr, and fills the blocks. A block after
  // the first merges the read with its samples. The first block takes the read only
  // while its coverage ends at the read's start. The result holds the requests to fetch
  // again: a first block that a new data type reset, a first catch-up block that
  // another fetch filled with newest samples it did not trust, and a block whose
  // history changed while the read ran.
  private async load(
    { first, last, start, catchUp, entries }: Group,
    tr: TimeRange,
  ): Promise<Set<Request>> {
    const { readRemote, retrieveChannels, store, now } = this.props;
    const committed = now().sub(COMMIT_MARGIN);
    const requested = Map.groupBy(
      entries.map(({ req }) => req),
      (req) => req.block.channel,
    );
    const channels = [...requested.keys()];
    const pins = entries.flatMap(({ req }) => (req.pinned == null ? [] : [req.pinned]));
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
      store.setChannel(p);
    });
    channels.forEach((ch) => {
      if (!indexOf.has(ch)) throw new NotFoundError(`channel ${ch} not found`);
    });
    // rewritten holds the channels whose history changed over the read's blocks while
    // it ran, and null when every channel's did.
    const rewritten = new Set<channel.Key | null>();
    const span = new TimeRange(
      tileRange({ level: RAW_LEVEL, index: first }).start,
      tr.end,
    );
    const stopWatching = store.onInvalidate((ch, changed) => {
      if (changed.overlapsWith(span)) rewritten.add(ch);
    });
    let frame: Frame;
    try {
      frame = await this.timed(
        readRemote(tr, [...indexOf.keys()]),
        `block read for ${tr.toString()}`,
      );
    } finally {
      stopWatching();
    }
    if (this.closed) return refetch;
    const stale = (k: channel.Key): boolean => rewritten.has(null) || rewritten.has(k);
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
      const parts = split(timestamps, series);
      const targets = [...(requested.get(key) ?? [])];
      // A read fills the blocks of the index it reads along with a channel, and a
      // catch-up fills the blocks of the newer samples it read.
      const extra = [...parts.keys()].filter((i) => i > last);
      if (targets.length === 0) for (let i = first; i <= last; i++) extra.push(i);
      extra.forEach((i) =>
        targets.push({ block: store.raw(key, i), end: tr.end, catchUp }),
      );
      const trusted = catchUp ? newestEnd([...parts.values()], start) : committed;
      // The Core stamps a series with the span of its domain inside the read, so the
      // channel's samples reach seen.
      const seen = series.reduce(
        (end, s) => TimeStamp.max(end, s.timeRange.end),
        TimeStamp.ZERO,
      );
      const rewrote = stale(key) || (idx !== 0 && stale(idx));
      targets.forEach((r) => {
        const { block } = r;
        if (block.joined) return;
        if (rewrote) {
          if (requested.has(key)) refetch.add(r);
          return;
        }
        const from = block.index === first ? start : block.range.start;
        if (changed) {
          block.expire();
          if (!from.equals(block.range.start)) {
            refetch.add(r);
            return;
          }
        }
        const merged = block.index !== first;
        if (!merged && !block.coveredEnd.equals(from)) {
          if (catchUp && requested.has(key)) refetch.add(r);
          return;
        }
        const part = parts.get(block.index) ?? { timestamps: [], series: [] };
        const end = TimeStamp.max(from, TimeStamp.min(block.range.end, trusted));
        const base = merged ? 0 : block.coveredCount;
        const aligned = align(block.index, base, part.timestamps, part.series);
        // An older read can end inside the coverage of a newer one, which then keeps
        // its end, count, and seal.
        if (block.coveredEnd.after(end))
          block.fill(aligned, block.coveredEnd, block.coveredCount);
        else {
          block.fill(aligned, end, base + countBefore(part.timestamps, end));
          block.sealed = !seen.before(end);
        }
        // A pin before the set, since the set can evict the blocks filled before it.
        pins.forEach((pinned) => {
          if (pinned.has(block)) return;
          pinned.add(block);
          store.repin(new Set(), new Set([block]));
        });
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

// Returns the end of the span of the newest sample in the parts, or start when there is
// none.
const newestEnd = (parts: Part[], start: TimeStamp): TimeStamp => {
  const part = parts.findLast((p) => p.series.length > 0);
  if (part == null) return start;
  const last = part.series[part.series.length - 1];
  const alignment = last.alignment + BigInt(last.length - 1);
  const ts = part.timestamps.find((t) => alignment < t.alignment + BigInt(t.length))!;
  return new TimeStamp(BigInt(ts.data[Number(alignment - ts.alignment)]) + 1n);
};

// Splits one channel's pending requests into runs of adjacent blocks that one Core read
// can fill: a run starts at the coverage end of its first block, and every later block
// in it has no coverage.
const runs = (entries: Array<scheduler.BatcherEntry<Request>>): Group[] => {
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
      run.end = TimeStamp.max(run.end, entry.req.end, range.end);
      run.entries.push(entry);
    } else
      out.push({
        first: index,
        last: index,
        start: coveredEnd,
        end: TimeStamp.max(entry.req.end, range.end),
        catchUp: entry.req.catchUp,
        entries: [entry],
      });
  }
  return out;
};
