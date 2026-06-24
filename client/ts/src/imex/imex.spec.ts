// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil/client";

const logEnvelope = (name: string) => ({
  version: 2,
  type: "log",
  name,
  channels: [
    {
      channel: 1,
      color: { r: 127, g: 29, b: 29, a: 1 },
      notation: "scientific",
      precision: 2,
      alias: "temp",
      timestamp: { format: "ISO", tz: "UTC" },
    },
  ],
  timestamp_precision: 1,
  hide_channel_names: false,
  hide_receipt_timestamp: true,
});

const toBlob = (value: unknown): Blob =>
  new Blob([JSON.stringify(value)], { type: "application/json" });

describe("Imex", () => {
  const client = createTestClient();

  it("should import from a Blob and export to a byte stream", async () => {
    const name = `imex-${id.create()}`;
    const oid = await client.imex.import(toBlob(logEnvelope(name)), {
      contentType: "json",
    });
    expect(oid.type).toEqual("log");
    expect(oid.key).not.toHaveLength(0);
    const stream = await client.imex.export(oid, { contentType: "json" });
    const parsed = await new Response(stream).json();
    expect(parsed.type).toEqual("log");
    expect(parsed.name).toEqual(name);
  });

  it("should import from raw bytes", async () => {
    const name = `imex-bytes-${id.create()}`;
    const bytes = new TextEncoder().encode(JSON.stringify(logEnvelope(name)));
    const oid = await client.imex.import(bytes, { contentType: "json" });
    expect(oid.type).toEqual("log");
    const exported = await new Response(
      await client.imex.export(oid, { contentType: "json" }),
    ).json();
    expect(exported.name).toEqual(name);
  });
});
