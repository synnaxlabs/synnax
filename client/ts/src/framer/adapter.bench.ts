// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series } from "@synnaxlabs/x";
import { allocSuiteAsync } from "@synnaxlabs/x/bench";
import { bench, describe } from "vitest";

import { type channel } from "@/channel";
import { payloadZ } from "@/channel/types.gen";
import { WriteAdapter } from "@/framer/adapter";
import { createKeys } from "@/framer/benchutil";
import { Frame } from "@/framer/frame";

const CHANNELS = 10;
const SAMPLES = 100;

const keys = createKeys(CHANNELS);
const payloads: channel.Payload[] = keys.map((key) =>
  payloadZ.parse({ key, name: `ch_${key}`, dataType: DataType.FLOAT32.toString() }),
);

const retrieveChannels = async (params: channel.Params): Promise<channel.Payload[]> => {
  const arr = Array.isArray(params) ? params : [params];
  return payloads.filter((p) =>
    arr.some((c) =>
      typeof c === "object" ? c.key === p.key : c === p.key || c === p.name,
    ),
  );
};

const adapter = await WriteAdapter.open(
  retrieveChannels,
  payloads.map((p) => p.name),
);

const data = new Float32Array(SAMPLES);
const record: Record<string, Float32Array> = Object.fromEntries(
  payloads.map((p) => [p.name, data]),
);
const keyedFrame = new Frame(
  keys,
  keys.map(() => new Series({ data, dataType: DataType.FLOAT32 })),
);

// The writer calls adapt on every write; a name-keyed record is the common
// ergonomic path in control code.
describe("write adapt", () => {
  bench("record by name", async () => {
    await adapter.adapt(record);
  });
  bench("frame by key", async () => {
    await adapter.adapt(keyedFrame);
  });
  bench("adapt + encode record", async () => {
    adapter.encode(await adapter.adapt(record));
  });
});

await allocSuiteAsync("write adapt 10ch x 100smp", [
  ["record by name", async () => void (await adapter.adapt(record))],
  [
    "adapt + encode record",
    async () => void adapter.encode(await adapter.adapt(record)),
  ],
]);
