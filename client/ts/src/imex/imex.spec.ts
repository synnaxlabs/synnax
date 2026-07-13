// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil";

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

const toBlob = (value: unknown): Blob => new Blob([JSON.stringify(value)]);

describe("Imex", () => {
  const client = createTestClient();

  describe("import", () => {
    it("should import from a Blob", async () => {
      const name = `imex-${id.create()}`;
      const ontologyID = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
      });
      expect(ontologyID.type).toEqual("log");
      expect(ontologyID.key).not.toHaveLength(0);
    });
    it("should throw an error if the envelope cannot be decoded", async () => {
      const envelope = logEnvelope("invalid");
      envelope.version = -1;
      await expect(
        client.imex.import(toBlob(envelope), { encoding: "JSON" }),
      ).rejects.toThrow("failed to decode");
    });
  });

  describe("export", () => {
    it("should export to a byte stream", async () => {
      const name = `imex-${id.create()}`;
      const oid = await client.imex.import(toBlob(logEnvelope(name)), {
        encoding: "JSON",
      });
      const stream = await client.imex.export(oid, { encoding: "JSON" });
      const parsed = await new Response(stream).json();
      expect(parsed.type).toEqual("log");
      expect(parsed.name).toEqual(name);
    });
    it("should throw an error if the log is not found", async () => {
      await expect(
        client.imex.export({ type: "log", key: uuid.create() }, { encoding: "JSON" }),
      ).rejects.toThrow("not found");
    });
  });
});
