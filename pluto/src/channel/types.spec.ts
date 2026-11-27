// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Haul } from "@synnaxlabs/lyra/haul";
import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Channel } from "@/channel";

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("channel haul utilities", () => {
  describe("createHaulItem", () => {
    it("creates an item with the channel HAUL_TYPE", () => {
      expect(Channel.createHaulItem(42).type).toEqual(Channel.HAUL_TYPE);
    });

    it("creates an item with the provided key", () => {
      expect(Channel.createHaulItem(42).key).toEqual(42);
    });
  });

  describe("isHaulItem", () => {
    it("returns true for an item of the channel kind", () => {
      expect(Channel.isHaulItem(Channel.createHaulItem(1))).toBe(true);
    });

    it("returns false for an item of another kind", () => {
      expect(Channel.isHaulItem(OTHER)).toBe(false);
    });
  });

  describe("filterHaulItems", () => {
    it("keeps channel items and drops items of other kinds", () => {
      const item = Channel.createHaulItem(1);
      expect(Channel.filterHaulItems([item, OTHER])).toEqual([item]);
    });
  });

  describe("canDropHaulItem", () => {
    it("returns true when at least one item is a channel item", () => {
      const item = Channel.createHaulItem(1);
      expect(Channel.canDropHaulItem({ source: OTHER, items: [item, OTHER] })).toBe(
        true,
      );
    });

    it("returns false when no item is a channel item", () => {
      expect(Channel.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(false);
    });
  });
});

describe("plottable", () => {
  const STORED = { dataType: DataType.FLOAT32, virtual: false, expression: "" };

  it("should plot a stored numeric channel", () => {
    expect(Channel.plottable(STORED)).toBe(true);
  });

  it("should plot a calculated numeric channel", () => {
    expect(
      Channel.plottable({ ...STORED, virtual: true, expression: "return 1.0" }),
    ).toBe(true);
  });

  it("should not plot a virtual channel without an expression", () => {
    expect(Channel.plottable({ ...STORED, virtual: true })).toBe(false);
  });

  it.each([DataType.STRING, DataType.JSON, DataType.BYTES, DataType.TIMESTAMP])(
    "should not plot a %s channel",
    (dataType) => {
      expect(Channel.plottable({ ...STORED, dataType })).toBe(false);
    },
  );
});

describe("filterPlottableHaulItems", () => {
  const NUMERIC = { dataType: DataType.FLOAT32, virtual: false, expression: "" };
  const TEXT = { ...NUMERIC, dataType: DataType.STRING };

  it("should keep a channel its source knows to be plottable", () => {
    const item = Channel.createHaulItem(1, NUMERIC);
    expect(Channel.filterPlottableHaulItems([item, OTHER])).toEqual([item]);
  });

  it("should drop a channel its source knows not to be plottable", () => {
    expect(Channel.filterPlottableHaulItems([Channel.createHaulItem(1, TEXT)])).toEqual(
      [],
    );
  });

  it("should keep a channel its source did not know", () => {
    const item = Channel.createHaulItem(1);
    expect(Channel.filterPlottableHaulItems([item])).toEqual([item]);
  });
});
