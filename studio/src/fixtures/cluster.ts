// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

/**
 * Seeding helpers for pre-staged cluster state. Shots should start mid-state
 * (existing ranges, users, historical data) rather than recording their own
 * setup; run these before startRecording against the capture's ephemeral core.
 */

export interface ClusterOptions {
  host?: string;
  port?: number;
  username?: string;
  password?: string;
}

/** defaultPort resolves the port of the capture's core, set by the pipeline. */
export const defaultPort = (): number => Number(process.env.SYNNAX_STUDIO_PORT ?? 9090);

const connect = ({
  host = "localhost",
  port = defaultPort(),
  username = "synnax",
  password = "seldon",
}: ClusterOptions): Synnax => new Synnax({ host, port, username, password });

export interface RangeSpec {
  name: string;
  /** Minutes before now the range ends (default staggers specs into the past). */
  endOffsetMin?: number;
  /** Range length in minutes. */
  durationMin?: number;
}

/**
 * seedRanges creates named ranges in the recent past. Offsets keep them clear
 * of a live plot's rolling window, so they only render as annotations when a
 * shot frames their span deliberately.
 */
export const seedRanges = async (
  specs: (string | RangeSpec)[],
  opts: ClusterOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    const now = TimeStamp.now();
    await client.ranges.create(
      specs.map((raw, i) => {
        const spec = typeof raw === "string" ? { name: raw } : raw;
        const end = now.sub(TimeSpan.minutes(spec.endOffsetMin ?? 30 + i * 20));
        const start = end.sub(TimeSpan.minutes(spec.durationMin ?? 10));
        return { name: spec.name, timeRange: new TimeRange(start, end) };
      }),
    );
  } finally {
    client.close();
  }
};

export interface UserSpec {
  username: string;
  password?: string;
  firstName?: string;
  lastName?: string;
}

/** seedUsers registers users on the cluster so user-management shots have rows. */
export const seedUsers = async (
  specs: UserSpec[],
  opts: ClusterOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    await client.users.create(specs.map((s) => ({ password: "seldon-demo", ...s })));
  } finally {
    client.close();
  }
};

export interface StaticTelemetryOptions extends ClusterOptions {
  /** Data channel names to create (each gets its own waveform). */
  channels?: string[];
  /** Sample count per channel. */
  samples?: number;
  /** Milliseconds between samples. */
  periodMs?: number;
}

/**
 * seedStaticTelemetry writes a block of historical waveform data ending just
 * before now, for shots that plot over a fixed range instead of streaming.
 * Returns the data channel names for selection in scripts.
 */
export const seedStaticTelemetry = async ({
  channels = ["demo_pressure", "demo_temperature"],
  samples = 3000,
  periodMs = 40,
  ...opts
}: StaticTelemetryOptions = {}): Promise<string[]> => {
  const client = connect(opts);
  try {
    const time = await client.channels.create(
      { name: "demo_time", isIndex: true, dataType: "timestamp" },
      { retrieveIfNameExists: true },
    );
    const data = await client.channels.create(
      channels.map((name) => ({ name, dataType: "float32", index: time.key })),
      { retrieveIfNameExists: true },
    );

    const span = TimeSpan.milliseconds(periodMs * samples);
    const start = TimeStamp.now().sub(span).sub(TimeSpan.seconds(5));
    const times = new BigInt64Array(samples);
    const step = BigInt(TimeSpan.milliseconds(periodMs).valueOf());
    for (let i = 0; i < samples; i++)
      times[i] = BigInt(start.valueOf()) + BigInt(i) * step;

    const frame: Record<
      string,
      BigInt64Array<ArrayBuffer> | Float32Array<ArrayBuffer>
    > = { [time.key]: times };
    data.forEach((ch, c) => {
      const wave = new Float32Array(samples);
      for (let i = 0; i < samples; i++)
        wave[i] = 60 + 25 * Math.sin(i / (25 + 10 * c) + c) + 2 * Math.sin(i / (3 + c));
      frame[ch.key] = wave;
    });
    await client.write(start, frame);
    return data.map((ch) => ch.name);
  } finally {
    client.close();
  }
};
