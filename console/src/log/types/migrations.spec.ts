// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { anyStateZ, migrateSlice, migrateState } from "@/log/types";
import * as v0 from "@/log/types/v0";
import * as v1 from "@/log/types/v1";
import * as v2 from "@/log/types/v2";

describe("log type migrations", () => {
  describe("identity", () => {
    it("should leave a v2 state unchanged", () => {
      expect(migrateState(v2.ZERO_STATE)).toEqual(v2.ZERO_STATE);
    });

    it("should stamp the latest version on every input", () => {
      [v0.ZERO_STATE, v1.ZERO_STATE, v2.ZERO_STATE].forEach((state) =>
        expect(migrateState(state).version).toBe(v2.VERSION),
      );
    });
  });

  describe("body -> pendingUpload", () => {
    it("should park a v0 body in pendingUpload with default channel config", () => {
      const migrated = migrateState({
        ...v0.ZERO_STATE,
        key: "test",
        channels: [1, 2],
        remoteCreated: false,
      });
      expect(migrated.pendingUpload?.channels).toEqual([
        log.channelEntryZ.parse({ channel: 1, color: color.ZERO }),
        log.channelEntryZ.parse({ channel: 2, color: color.ZERO }),
      ]);
    });

    it("should park a v1 body in pendingUpload, preserving display config", () => {
      const migrated = migrateState({
        ...v1.ZERO_STATE,
        key: "test",
        remoteCreated: false,
        timestampPrecision: 2,
        showChannelNames: false,
        channels: [
          v1.channelEntryZ.parse({ channel: 1, color: "#ff0000", precision: 3 }),
        ],
      });
      expect(migrated.pendingUpload).toMatchObject({
        key: "test",
        timestampPrecision: 2,
        showChannelNames: false,
      });
      expect(migrated.pendingUpload?.channels[0]).toMatchObject({
        channel: 1,
        precision: 3,
        color: color.construct("#ff0000"),
      });
    });

    it("should leave pendingUpload undefined for a remoteCreated log", () => {
      const migrated = migrateState({
        ...v1.ZERO_STATE,
        key: "test",
        remoteCreated: true,
        channels: [v1.channelEntryZ.parse({ channel: 1 })],
      });
      expect(migrated.pendingUpload).toBeUndefined();
    });

    it("should preserve the toolbar tab through migration", () => {
      const migrated = migrateState({
        ...v1.ZERO_STATE,
        key: "test",
        toolbar: { activeTab: "properties" },
      });
      expect(migrated.toolbar.activeTab).toBe("properties");
    });
  });

  describe("v1 color migration into pendingUpload", () => {
    const migratedColor = (channelColor: string) =>
      migrateState({
        ...v1.ZERO_STATE,
        key: "test",
        remoteCreated: false,
        channels: [v1.channelEntryZ.parse({ channel: 1, color: channelColor })],
      }).pendingUpload?.channels[0].color;

    it("should convert an empty-string color to color.ZERO", () => {
      expect(migratedColor("")).toEqual(color.ZERO);
    });

    it("should convert a hex string to a real color", () => {
      expect(migratedColor("#ff0000")).toEqual(color.construct("#ff0000"));
    });

    it("should fall back to color.ZERO for an unparseable color", () => {
      expect(migratedColor("not-a-color")).toEqual(color.ZERO);
    });
  });

  describe("slice", () => {
    it("should migrate every log in a v1 slice", () => {
      const migrated = migrateSlice({
        ...v1.ZERO_SLICE_STATE,
        logs: {
          test: {
            ...v1.ZERO_STATE,
            key: "test",
            remoteCreated: false,
            channels: [v1.channelEntryZ.parse({ channel: 1, color: "" })],
          },
        },
      });
      expect(migrated.version).toBe(v2.VERSION);
      expect(migrated.logs.test.pendingUpload?.channels[0].color).toEqual(color.ZERO);
    });
  });

  describe("anyStateZ", () => {
    it("should parse a v2 state as-is", () => {
      expect(anyStateZ.parse(v2.ZERO_STATE).version).toBe(v2.VERSION);
    });

    it("should parse and migrate a v0 state", () => {
      expect(anyStateZ.parse(v0.ZERO_STATE).version).toBe(v2.VERSION);
    });

    it("should parse a persisted v1 log export into a pendingUpload", () => {
      const result = anyStateZ.parse({
        key: "424ef02f-6ec3-4af6-bdd2-242964a747d8",
        version: "1.0.0",
        channels: [
          {
            channel: 1048581,
            color: "",
            notation: "standard",
            precision: -1,
            alias: "",
          },
        ],
        remoteCreated: false,
        timestampPrecision: 0,
        showChannelNames: true,
        showReceiptTimestamp: true,
        toolbar: { activeTab: "channels" },
        type: "log",
      });
      expect(result.version).toBe(v2.VERSION);
      expect(result.pendingUpload?.channels[0].color).toEqual(color.ZERO);
      expect(result.pendingUpload?.channels[0].timestamp).toEqual({
        format: "preciseDate",
        tz: "local",
      });
    });
  });
});
