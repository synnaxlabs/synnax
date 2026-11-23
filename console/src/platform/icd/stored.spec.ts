// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

describe("ICD", () => {
  describe("bindingsOf", () => {
    const [a, a2, b, c, g] = Array.from({ length: 5 }, () => uuid.create());
    const dataTypes = new Map<channel.Key, string>([
      [1, "float64"],
      [2, "string"],
    ]);
    const channelsOf = (
      fields: { key: string; pointer: string; type: "float" | "string" }[],
      written = new Set<channel.Key>(),
    ) =>
      ICD.bindingsOf({ "/a": 1, "/b": 2, "/gone": 3 }, fields, written, dataTypes).map(
        ({ field, channel }) => [field, channel],
      );

    it("should bind each field to the channel stored for its pointer", () => {
      const written = new Set<channel.Key>();
      expect(
        channelsOf(
          [
            { key: a, pointer: "/a", type: "float" },
            { key: b, pointer: "/b", type: "string" },
          ],
          written,
        ),
      ).toEqual([
        [a, 1],
        [b, 2],
      ]);
      expect(written).toEqual(new Set([1, 2]));
    });

    it("should bind no channel for a pointer with no stored channel", () => {
      expect(channelsOf([{ key: c, pointer: "/c", type: "float" }])).toEqual([[c, 0]]);
    });

    it("should bind no channel that another binding writes", () => {
      expect(
        channelsOf(
          [
            { key: a, pointer: "/a", type: "float" },
            { key: a2, pointer: "/a", type: "float" },
          ],
          new Set(),
        ),
      ).toEqual([
        [a, 1],
        [a2, 0],
      ]);
      expect(
        channelsOf([{ key: a, pointer: "/a", type: "float" }], new Set([1])),
      ).toEqual([[a, 0]]);
    });

    it("should bind no channel whose data type the kind does not convert to", () => {
      expect(channelsOf([{ key: a, pointer: "/a", type: "string" }])).toEqual([[a, 0]]);
    });

    it("should bind no channel whose data type is unknown", () => {
      expect(channelsOf([{ key: g, pointer: "/gone", type: "float" }])).toEqual([
        [g, 0],
      ]);
    });
  });
});
