// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";

import { connect, type ConnectionOptions } from "@/fixtures/client";

/**
 * Helpers for pre-staged cluster state. Shots should start mid-state (existing
 * ranges, users, historical data) rather than recording their own setup; run
 * these before startRecording against the capture's ephemeral core.
 */

export interface RangeSpec {
  name: string;
  /** Minutes before now the range ends (default staggers specs into the past). */
  endOffsetMin?: number;
  /** Range length in minutes. */
  durationMin?: number;
  /** Absolute span; overrides the now-relative offsets when set. */
  timeRange?: TimeRange;
}

/**
 * createRanges creates named ranges in the recent past. Offsets keep them clear
 * of a live plot's rolling window, so they only render as annotations when a
 * shot frames their span deliberately.
 */
export const createRanges = async (
  specs: (string | RangeSpec)[],
  opts: ConnectionOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    const now = TimeStamp.now();
    await client.ranges.create(
      specs.map((raw, i) => {
        const spec = typeof raw === "string" ? { name: raw } : raw;
        if (spec.timeRange != null)
          return { name: spec.name, timeRange: spec.timeRange };
        const end = now.sub(TimeSpan.minutes(spec.endOffsetMin ?? 30 + i * 20));
        const start = end.sub(TimeSpan.minutes(spec.durationMin ?? 10));
        return { name: spec.name, timeRange: new TimeRange(start, end) };
      }),
    );
  } finally {
    await client.close();
  }
};

export interface LabelSpec {
  name: string;
  color: string;
}

/** createLabels creates labels on the cluster for label-selection shots. */
export const createLabels = async (
  specs: LabelSpec[],
  opts: ConnectionOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    await client.labels.create(specs);
  } finally {
    await client.close();
  }
};

/**
 * resetLabels leaves the cluster holding exactly the given labels: a listed
 * name is recolored to match, every other label and any duplicate is deleted,
 * and a missing listed label is created. Keeps a clean, known label set.
 */
export const resetLabels = async (
  specs: LabelSpec[],
  opts: ConnectionOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    const want = new Map(specs.map((s) => [s.name, s.color]));
    const kept = new Set<string>();
    const remove: string[] = [];
    for (const l of await client.labels.retrieve({ limit: 1000 })) {
      const color = want.get(l.name);
      if (color != null && !kept.has(l.name)) {
        kept.add(l.name);
        await client.labels.create({ key: l.key, name: l.name, color });
      } else remove.push(l.key);
    }
    if (remove.length > 0) await client.labels.delete(remove);
    const missing = specs.filter((s) => !kept.has(s.name));
    if (missing.length > 0) await client.labels.create(missing);
  } finally {
    await client.close();
  }
};

/**
 * ensureLabels creates any listed label missing by name, leaving existing ones
 * untouched, and returns a name-to-key map for assigning them to resources.
 */
export const ensureLabels = async (
  specs: LabelSpec[],
  opts: ConnectionOptions = {},
): Promise<Record<string, string>> => {
  const client = connect(opts);
  try {
    const byName = new Map(
      (await client.labels.retrieve({ limit: 1000 })).map((l) => [l.name, l.key]),
    );
    const keys: Record<string, string> = {};
    for (const spec of specs)
      keys[spec.name] = byName.get(spec.name) ?? (await client.labels.create(spec)).key;
    return keys;
  } finally {
    await client.close();
  }
};

export interface RangeSetup {
  name: string;
  timeRange: TimeRange;
  /** Fixed key; lists that follow key order can be sequenced through it. */
  key?: string;
  /** Parent range key, for a child range. */
  parent?: string;
  /** Label keys to attach. */
  labels?: string[];
}

/**
 * ensureRange creates the range if none shares its name, attaches any labels,
 * and returns its key. Skips creation when it already exists, so a run leaves
 * an existing tree in place.
 */
export const ensureRange = async (
  spec: RangeSetup,
  opts: ConnectionOptions = {},
): Promise<string> => {
  const client = connect(opts);
  try {
    const all = await client.ranges.retrieve({ limit: 1000 }).catch(() => []);
    const range =
      all.find((r) => r.name === spec.name) ??
      (await client.ranges.create({
        name: spec.name,
        timeRange: spec.timeRange,
        ...(spec.key != null && { key: spec.key }),
        ...(spec.parent != null && { parent: { key: spec.parent } }),
      }));
    if (spec.labels != null && spec.labels.length > 0)
      await client.labels.label(range.ontologyID, spec.labels);
    return range.key;
  } finally {
    await client.close();
  }
};

/**
 * clearWorkspace deletes every range and view on the cluster, leaving labels
 * and the rest untouched, so a shot starts and ends from a bare workspace.
 */
export const clearWorkspace = async (opts: ConnectionOptions = {}): Promise<void> => {
  const client = connect(opts);
  try {
    const ranges = await client.ranges.retrieve({ limit: 1000 });
    if (ranges.length > 0) await client.ranges.delete(ranges.map((r) => r.key));
    const views = await client.views.retrieve({ limit: 1000 });
    if (views.length > 0) await client.views.delete(views.map((v) => v.key));
  } finally {
    await client.close();
  }
};

/** removeChannels deletes the named channels, so a shot can create them on camera. */
export const removeChannels = async (
  names: string[],
  opts: ConnectionOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    const found = await client.channels.retrieve({ names });
    if (found.length > 0) await client.channels.delete(found.map((c) => c.key));
  } finally {
    await client.close();
  }
};

export interface CalculatedSpec {
  name: string;
  /** Arc expression, e.g. `return demo_pressure * 2`. */
  expression: string;
}

/**
 * createCalculatedChannels creates virtual calculated channels, for shots that
 * start from one that already exists.
 */
export const createCalculatedChannels = async (
  specs: CalculatedSpec[],
  opts: ConnectionOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    await client.channels.create(
      specs.map((spec) => ({ ...spec, virtual: true, dataType: "float32" })),
    );
  } finally {
    await client.close();
  }
};

export interface UserSpec {
  username: string;
  password?: string;
  firstName?: string;
  lastName?: string;
  /** Name of a built-in role to assign; a user without one stays hidden. */
  role?: string;
}

/** createUsers registers users on the cluster so user-management shots have rows. */
export const createUsers = async (
  specs: UserSpec[],
  opts: ConnectionOptions = {},
): Promise<void> => {
  const client = connect(opts);
  try {
    const users = await client.users.create(
      specs.map(({ role: _role, ...s }) => ({ password: "seldon-demo", ...s })),
    );
    const roles = await client.access.roles.retrieve({});
    await Promise.all(
      specs.map(async ({ role }, i) => {
        if (role == null) return;
        const match = roles.find((r) => r.name === role);
        if (match == null) throw new Error(`role ${role} not found`);
        await client.access.roles.assign({ role: match.key, user: users[i].key });
      }),
    );
  } finally {
    await client.close();
  }
};

export interface StaticTelemetryOptions extends ConnectionOptions {
  /** Data channel names to create (each gets its own waveform). */
  channels?: string[];
  /** Sample count per channel. */
  samples?: number;
  /** Milliseconds between samples. */
  periodMs?: number;
}

/**
 * createStaticTelemetry writes a block of historical waveform data ending just
 * before now, for shots that plot over a fixed range instead of streaming.
 * Returns the data channel names for selection in scripts.
 */
export const createStaticTelemetry = async ({
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
    await client.close();
  }
};
