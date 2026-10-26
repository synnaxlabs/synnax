// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Unreachable } from "@synnaxlabs/freighter";
import { type breaker, Rate, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type channel } from "@/channel";
import { Joins } from "@/framer/cache/join";
import { Joiner } from "@/framer/cache/joiner";
import { Reader, type RemoteReader } from "@/framer/cache/reader";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer } from "@/framer/cache/streamer";
import { SyntheticCore, type SyntheticCoreProps } from "@/testutil";

const START = TimeStamp.seconds(1_000_000);
const POLL = { timeout: 5000, interval: 5 };
const BREAKER: breaker.Config = {
  baseInterval: TimeSpan.milliseconds(10),
  maxInterval: TimeSpan.milliseconds(50),
};

const createCore = (props: Partial<SyntheticCoreProps> = {}): SyntheticCore =>
  new SyntheticCore({
    rate: Rate.hz(100),
    start: START,
    now: START.add(TimeSpan.minutes(10)),
    ...props,
  });

interface JoinerOptions {
  breaker?: breaker.Config;
  readRemote?: RemoteReader;
}

// Wires a joiner to the Core as a feed does.
const createJoiner = (
  core: SyntheticCore,
  {
    breaker = BREAKER,
    readRemote = async (tr, keys, reduction) => core.read(tr, keys, reduction),
  }: JoinerOptions = {},
) => {
  const store = new Store({ now: core.now });
  const joins = new Joins(store);
  const reader = new Reader({
    readRemote,
    retrieveChannels: core.retrieveChannels,
    store,
    batchDebounce: TimeSpan.milliseconds(5),
    now: core.now,
  });
  const streamer = new MultiplexedStreamer({
    store,
    joins,
    onWait: (indexes) => joiner.wake(indexes),
    openStreamer: async (config) => core.openStreamer(config),
    removalDelay: TimeSpan.ZERO,
    breaker,
  });
  const joiner = new Joiner({ joins, reader, streamer, now: core.now, breaker });
  // Streams the channel with its index and joins them, as a subscription does.
  const join = (
    ch: channel.Key,
    index: channel.Key,
    landed: () => void = () => {},
  ): (() => void) => {
    const demand = streamer.demand([ch, index], () => {});
    const leave = joiner.join(ch, index, landed);
    return () => {
      leave();
      demand.close();
    };
  };
  const close = async (): Promise<void> => {
    joiner.close();
    await streamer.close();
    await reader.close();
    joins.close();
    store.close();
  };
  return { store, joins, joiner, join, close };
};

// Advances the Core's clock by 50ms on each tick of the real clock. Returns a function
// that stops it.
const tick = (core: SyntheticCore): (() => void) => {
  const id = setInterval(() => core.advance(TimeSpan.milliseconds(50)), 2);
  return () => clearInterval(id);
};

describe("Joiner", () => {
  it("should land a member once the stream acks it and its index", async () => {
    const core = createCore();
    const [index, data] = [core.index.key, core.data[0].key];
    const { store, joins, join, close } = createJoiner(core);
    let landings = 0;
    join(data, index, () => landings++);
    const stop = tick(core);
    try {
      await expect
        .poll(() => store.open(data)?.coveredCount ?? 0, POLL)
        .toBeGreaterThan(0);
      expect(joins.joined(data)).toBe(true);
      expect(landings).toBeGreaterThan(0);
      expect(store.open(data)?.coveredCount).toBe(store.open(index)?.coveredCount);
    } finally {
      stop();
      await close();
    }
  });

  it("should keep the rest of a join going when the Core rejects one member", async () => {
    const core = createCore({ channels: 2 });
    const index = core.index.key;
    const [allowed, denied] = core.data.map((c) => c.key);
    core.deny(denied);
    const read: channel.Key[] = [];
    const { store, joiner, join, close } = createJoiner(core, {
      readRemote: async (tr, keys, reduction) => {
        read.push(...keys);
        return core.read(tr, keys, reduction);
      },
    });
    join(allowed, index);
    join(denied, index);
    const stop = tick(core);
    try {
      await expect
        .poll(() => joiner.status(denied), POLL)
        .toMatchObject({
          variant: "error",
          message: "Failed to read channel data",
          description: `You do not have permission to do that. access to channel ${denied} denied`,
        });
      // A gap sends the whole join back to wait, and the allowed member lands again.
      core.advance(TimeSpan.seconds(2), false);
      const gap = core.now();
      await expect
        .poll(() => store.open(allowed)?.coveredEnd.after(gap) ?? false, POLL)
        .toBe(true);
      expect(joiner.status(allowed)).toBeUndefined();
      expect(read).not.toContain(denied);
    } finally {
      stop();
      await close();
    }
  });

  it("should clear the failure of a member when it leaves", async () => {
    const core = createCore();
    const [index, data] = [core.index.key, core.data[0].key];
    core.deny(data);
    const { joiner, join, close } = createJoiner(core);
    const leave = join(data, index);
    try {
      await expect.poll(() => joiner.status(data)?.variant, POLL).toBe("error");
      leave();
      expect(joiner.status(data)).toBeUndefined();
    } finally {
      await close();
    }
  });

  it("should back off a member that each landing fails to join", async () => {
    const core = createCore({ dataMisaligned: true });
    const [index, data] = [core.index.key, core.data[0].key];
    let reads = 0;
    const { join, close } = createJoiner(core, {
      breaker: {
        baseInterval: TimeSpan.milliseconds(20),
        maxInterval: TimeSpan.seconds(1),
      },
      readRemote: async (tr, keys, reduction) => {
        if (keys.includes(data)) reads++;
        return core.read(tr, keys, reduction);
      },
    });
    join(data, index);
    const stop = tick(core);
    try {
      await expect.poll(() => reads, POLL).toBeGreaterThan(1);
      // Waits of 20, 40, 80, 160, and 320ms leave room for about six reads.
      await sleep.sleep(TimeSpan.milliseconds(600));
      expect(reads).toBeLessThanOrEqual(8);
    } finally {
      stop();
      await close();
    }
  });

  it("should retry a failed read until it lands", async () => {
    const core = createCore();
    const [index, data] = [core.index.key, core.data[0].key];
    let failures = 3;
    const { joins, joiner, join, close } = createJoiner(core, {
      readRemote: async (tr, keys, reduction) => {
        if (keys.includes(data) && failures-- > 0)
          throw new Unreachable({ message: "core unreachable" });
        return core.read(tr, keys, reduction);
      },
    });
    join(data, index);
    const stop = tick(core);
    try {
      await expect.poll(() => joins.joined(data), POLL).toBe(true);
      expect(failures).toBeLessThan(0);
      expect(joiner.status(data)).toBeUndefined();
    } finally {
      stop();
      await close();
    }
  });

  it("should retry a calculation's read that fails for another reason", async () => {
    const core = createCore({ derived: true });
    const { key: calculated, index } = core.calculated!;
    let failures = 1;
    const { joins, joiner, join, close } = createJoiner(core, {
      readRemote: async (tr, keys, reduction) => {
        if (keys.includes(calculated) && failures-- > 0)
          throw new Error("stream closed unexpectedly");
        return core.read(tr, keys, reduction);
      },
    });
    join(calculated, index);
    const stop = tick(core);
    try {
      await expect.poll(() => joins.joined(calculated), POLL).toBe(true);
      expect(failures).toBeLessThan(0);
      expect(joiner.status(calculated)).toBeUndefined();
    } finally {
      stop();
      await close();
    }
  });

  it("should join a calculation and its index without history when reads fail", async () => {
    const core = createCore({ derived: true });
    const { key: calculated, index } = core.calculated!;
    const { store, joins, joiner, join, close } = createJoiner(core, {
      readRemote: async (tr, keys, reduction) => {
        if (keys.includes(calculated) || keys.includes(index))
          throw new Error("cannot open iterator on virtual channel");
        return core.read(tr, keys, reduction);
      },
    });
    join(calculated, index);
    const stop = tick(core);
    try {
      await expect
        .poll(() => joiner.status(calculated), POLL)
        .toMatchObject({
          variant: "warning",
          message: "Failed to read calculated channel history",
        });
      await expect
        .poll(() => store.open(calculated)?.coveredCount ?? 0, POLL)
        .toBeGreaterThan(0);
      expect(joins.joined(calculated)).toBe(true);
    } finally {
      stop();
      await close();
    }
  });
});
