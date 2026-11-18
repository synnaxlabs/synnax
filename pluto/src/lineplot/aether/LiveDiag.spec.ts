// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// [DIAG] Throwaway: repeats the live writer spec and dumps the cache on a miss.

import { createTestClient, TEST_CLIENT_PARAMS } from "@synnaxlabs/client/testutil";
import { box, DataType, dynamicWindow, id, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";

import { aetherTest } from "@/aether/testutil";
import { LinePlot } from "@/lineplot/aether/LinePlot";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { telem } from "@/telem/aether";
import { buildStack } from "@/testutil/providers";
import { line } from "@/vis/line/aether";
import { lineTest } from "@/vis/line/aether/testutil";

/* eslint-disable @typescript-eslint/no-explicit-any */

const REPEATS = Number(process.env.DIAG_REPEATS ?? (process.env.DIAG_FORCE == null ? 40 : 1));

const replacer = (_: string, v: unknown): unknown =>
  typeof v === "bigint" ? v.toString() : v;

const dumpSeries = (s: any): unknown => {
  try {
    return {
      a: s.alignment?.toString(),
      n: s.length,
      tr: s.timeRange?.toString(),
      off: String(s.sampleOffset),
      first: s.length > 0 ? String(s.at(0)) : null,
      last: s.length > 0 ? String(s.at(-1)) : null,
    };
  } catch (e) {
    return String(e);
  }
};

const channelDataOf = (src: any): any => {
  let o = src;
  for (let i = 0; i < 5 && o != null; i++) {
    if ("draw" in o) return o;
    o = o.wrapped;
  }
  return null;
};

const dumpPoints = (p: any): unknown => {
  try {
    return {
      key: p.key,
      index: p.index,
      stored: String(p.stored),
      parked: p.parked?.message,
      readFailure: p.readFailure?.message,
      rejection: p.rejection?.message,
      rawRead: p.rawRead,
      loading: p.loading,
      boundary: p.boundary?.toString(),
      pending: p.pending?.size,
      held: p.held?.size,
      live: p.live?.keys,
      grown: p.grown,
      memo: p.memo == null ? null : { epoch: p.memo.epoch, until: String(p.memo.validUntil) },
      epoch: p.props?.store?.epoch(p.key),
      limit: JSON.stringify(p.limit, replacer),
      raw: (p.raw ?? []).map(dumpSeries),
    };
  } catch (e) {
    return String(e);
  }
};

const dumpBlock = (b: any): unknown => {
  try {
    return {
      ch: b.channel,
      i: b.index,
      range: b.range?.toString(),
      covStart: b.coveredStart?.toString(),
      covEnd: b.coveredEnd?.toString(),
      covCount: b.coveredCount,
      joined: b.joined,
      sealed: b.sealed,
      pins: b.pins,
      fetched: (b.fetched ?? []).map(dumpSeries),
      lead: b.buffer?.leadingBuffer == null ? null : dumpSeries(b.buffer.leadingBuffer),
      dataRange: b.buffer?.dataTimeRange?.toString(),
    };
  } catch (e) {
    return String(e);
  }
};

const dumpFeed = (feed: any, keys: number[]): unknown => {
  try {
    const joins = feed.joins;
    const members = keys.map((k) => {
      const m = joins.members.get(k);
      if (m == null) return { k, member: null };
      return {
        k,
        joined: m.joined,
        wait: m.wait,
        from: m.from,
        fresh: m.fresh,
        historyless: m.historyless,
        held: m.held.map(dumpSeries),
        indexJoined: m.join.indexMember.joined,
        recent: m.join.recent.length,
        last: m.join.last == null ? null : dumpSeries(m.join.last),
      };
    });
    const blocks: unknown[] = [];
    feed.store.entries.forEach((e: any, key: string) => {
      if (keys.includes(e.block?.channel) && e.block?.coveredCount != null)
        blocks.push({ key, ...(dumpBlock(e.block) as object) });
    });
    const drivers: unknown[] = [];
    feed.joiner.drivers.forEach((d: any, k: number) =>
      drivers.push({
        k,
        running: d.running,
        landed: [...d.landed],
        rejected: [...d.rejected],
      }),
    );
    const failures: unknown[] = [];
    feed.joiner.failures.forEach((s: any, k: number) =>
      failures.push({ k, message: s.message, description: s.description }),
    );
    return {
      members,
      blocks,
      drivers,
      failures,
      acked: [...feed.streamer.acked],
      sent: [...feed.streamer.sentKeys],
      streaming: feed.streamer.streamer != null,
    };
  } catch (e) {
    return String(e);
  }
};

describe("[DIAG] live writer", () => {
  const synnax = createTestClient();
  const teardowns: (() => void)[] = [];
  afterEach(() => {
    for (const teardown of teardowns) teardown();
    teardowns.length = 0;
  });

  const mount = (props: telem.ChannelDataProps) => {
    const stack = buildStack({
      registry: {
        [telem.PROVIDER_TYPE]: telem.Provider,
        [lineTest.Host.TYPE]: lineTest.Host,
        [LinePlot.TYPE]: LinePlot,
        [XAxis.TYPE]: XAxis,
        [YAxis.TYPE]: YAxis,
        [line.Line.TYPE]: line.Line,
      },
      synnax: { props: TEST_CLIENT_PARAMS },
      render: true,
    });
    const gl = [...stack.basePath, "gl"];
    stack.driver.update(gl, lineTest.Host.TYPE, {});
    const base = [...gl, "plot"];
    stack.driver.update(base, LinePlot.TYPE, {
      container: box.construct(0, 0, 1000, 500),
      viewport: box.DECIMAL,
      grid: {},
    });
    stack.driver.update([...base, "x1"], XAxis.TYPE, { location: "bottom" });
    stack.driver.update([...base, "x1", "y1"], YAxis.TYPE, { location: "left" });
    stack.driver.update([...base, "x1", "y1", "l1"], line.Line.TYPE, {
      x: telem.channelData({ ...props, useIndexOfChannel: true }),
      y: telem.channelData(props),
      color: "#ff0000",
    });
    teardowns.push(() => stack.driver.delete([aetherTest.ROOT_KEY]));
    return { plot: stack.driver.find<LinePlot>(base) };
  };

  const gaps = (plot: LinePlot, targets: number[]): number[] =>
    targets.map((t) => {
      const [found] = plot.findByXValue(t);
      const x = found?.value.x ?? NaN;
      return Number.isNaN(x) ? Infinity : x - t;
    });

  const STARTS = (process.env.DIAG_STARTS ?? "").split(",").filter((v) => v !== "").map(Number);
  const cases = STARTS.length > 0 ? STARTS : Array.from({ length: REPEATS }, () => 11);
  for (let rep = 0; rep < cases.length; rep++)
    it(`should draw samples as a live writer commits them, ${rep}`, async () => {
      const startAt = cases[rep];
      let polling: ReturnType<typeof setInterval> | null = null;
      const t0 = performance.now();
      const events: string[] = [];
      const log = (e: string): void => {
        events.push(`${(performance.now() - t0).toFixed(1)} ${e}`);
      };
      const time = await synnax.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const data = await synnax.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        index: time.key,
      });
      const keys = [time.key, data.key];
      log(`channels ${time.key} ${data.key}`);
      const { plot } = mount({
        channel: data.key,
        window: dynamicWindow(TimeSpan.seconds(10)),
      });
      const lineOf = (): any => (plot as any).axes[0].yAxes[0].lines[0];
      let feed: any = null;
      const hook = (): void => {
        try {
          const l = lineOf();
          const cd = channelDataOf(l?.internal?.xTelem);
          const f = cd?.client?.feed;
          if (f == null || feed != null) return;
          feed = f;
          log("hooked");
          const wrap = (obj: any, name: string, fmt: (args: any[], r: any) => string) => {
            const orig = obj[name].bind(obj);
            obj[name] = (...args: any[]) => {
              const r = orig(...args);
              try {
                log(`${name} ${fmt(args, r)}`);
              } catch {}
              if (r instanceof Promise)
                r.then(
                  (v) => {
                    try {
                      log(`${name} settled ${fmt(args, v)}`);
                    } catch {}
                  },
                  (e) => log(`${name} rejected ${String(e)}`),
                );
              return r;
            };
          };
          wrap(feed.streamer, "setAcked", (a) => JSON.stringify([...a[0]]));
          wrap(feed.joins, "write", (a) =>
            JSON.stringify(
              [...a[0].entries()].map(([k, ss]: any) => [k, ss.map(dumpSeries)]),
            ),
          );
          wrap(feed.joins, "landed", (a) => JSON.stringify([a[0], a[1], a[2]]));
          wrap(feed.joins, "invalidate", (a) => `${a[0]} ${String(a[1])}`);
          wrap(feed.joins, "flush", (a) => String(a[0]));
          wrap(feed.reader, "load", (a, r) =>
            r instanceof Promise
              ? `${JSON.stringify({ first: a[0].first, last: a[0].last, catchUp: a[0].catchUp, keys: a[0].entries.map((e: any) => e.req.block.channel) })} ${a[1].toString()}`
              : "done",
          );
          wrap(feed.props, "readLatest", (a, r) =>
            r instanceof Promise
              ? JSON.stringify(a[0])
              : JSON.stringify(
                  r.keys?.map((k: number) => [k, r.get(k).series.map(dumpSeries)]),
                ),
          );
        } catch (e) {
          log(`hook failed ${String(e)}`);
        }
      };
      hook();
      const hooker = setInterval(hook, 1);
      const writer = await synnax.openWriter({
        start: TimeStamp.now(),
        channels: [time.key, data.key],
      });
      log("writer open");
      const step = TimeSpan.milliseconds(1).valueOf();
      let last = TimeStamp.now();
      try {
        for (let chunk = 0; chunk < 10; chunk++) {
          if (chunk === startAt && polling == null) {
            log("first read");
            gaps(plot, [0]);
            polling = setInterval(() => gaps(plot, [0]), 50);
          }
          const start = TimeStamp.now();
          const times = new BigInt64Array(100);
          const values = new Float32Array(100);
          for (let i = 0; i < 100; i++) {
            times[i] = start.valueOf() + BigInt(i) * step;
            values[i] = Math.sin(i / 10);
          }
          await writer.write({ [time.key]: times, [data.key]: values });
          log(`wrote ${chunk} ${times[0]}..${times[99]}`);
          last = new TimeStamp(times[99]);
          if (chunk === 9 && startAt === 10) {
            log("first read");
            gaps(plot, [0]);
            continue;
          }
          await sleep.sleep(TimeSpan.milliseconds(100));
        }
        if (polling != null) clearInterval(polling);
        clearInterval(hooker);
        const target =
          Number(last.valueOf()) + (process.env.DIAG_FORCE == null ? 0 : 5e6);
        try {
          await expect
            .poll(() => gaps(plot, [target])[0], {
              timeout: process.env.DIAG_FORCE == null ? 10_000 : 1_000,
            })
            .toBe(0);
          if (process.env.DIAG_EVENTS != null)
            process.stdout.write(`[DIAG-PASS] ${JSON.stringify(events)}\n`);
        } catch (e) {
          const l = lineOf();
          const x = channelDataOf(l?.internal?.xTelem);
          const y = channelDataOf(l?.internal?.yTelem);
          const valueOf = (cd: any) => {
            try {
              const v = cd.value({});
              return {
                bounds: JSON.stringify(v.data[0]),
                status: v.status?.variant,
                message: v.status?.message,
                description: v.status?.description,
                series: v.data[1].series.map(dumpSeries),
              };
            } catch (err) {
              return String(err);
            }
          };
          const pointsOf = (cd: any) => {
            const out: unknown[] = [];
            cd?.draw?.consumers?.forEach((p: any) => out.push(dumpPoints(p)));
            cd?.bounds?.consumers?.forEach((p: any) =>
              out.push({ bounds: true, ...(dumpPoints(p) as object) }),
            );
            return out;
          };
          const dump = {
            target: last.valueOf(),
            now: TimeStamp.now().valueOf(),
            found: JSON.stringify(plot.findByXValue(target), replacer),
            x: { value: valueOf(x), points: pointsOf(x) },
            y: { value: valueOf(y), points: pointsOf(y) },
            feed: dumpFeed(feed ?? x?.client?.feed, keys),
            events,
          };
          process.stdout.write(`[DIAG-DUMP] ${JSON.stringify(dump, replacer)}\n`);
          throw e;
        }
      } finally {
        if (polling != null) clearInterval(polling);
        clearInterval(hooker);
        await writer.close();
      }
    });
});
