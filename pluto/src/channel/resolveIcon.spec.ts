// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { DataType } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { resolveIcon } from "@/channel/resolveIcon";
import { Icon } from "@/icon";

describe("resolveIcon", () => {
  it("should return Icon.Channel when no channel is provided", () => {
    expect(resolveIcon()).toBe(Icon.Channel);
  });

  it("should return Icon.Channel when channel is undefined", () => {
    expect(resolveIcon(undefined)).toBe(Icon.Channel);
  });

  it("should return Icon.Calculation for a calculated channel", () => {
    const ch: channel.Payload = {
      key: 1,
      name: "calc",
      dataType: DataType.FLOAT32,
      isIndex: false,
      index: 0,
      leaseholder: 0,
      virtual: true,
      expression: "x + 1",
      internal: false,
      operations: [],
    };
    expect(resolveIcon(ch)).toBe(Icon.Calculation);
  });

  it("should return Icon.Decimal for a float channel", () => {
    const ch: channel.Payload = {
      key: 1,
      name: "temp",
      dataType: DataType.FLOAT32,
      isIndex: false,
      index: 0,
      leaseholder: 0,
      virtual: false,
      expression: "",
      internal: false,
      operations: [],
    };
    expect(resolveIcon(ch)).toBe(Icon.Decimal);
  });

  it("should return Icon.Binary for an integer channel", () => {
    const ch: channel.Payload = {
      key: 1,
      name: "count",
      dataType: DataType.INT32,
      isIndex: false,
      index: 0,
      leaseholder: 0,
      virtual: false,
      expression: "",
      internal: false,
      operations: [],
    };
    expect(resolveIcon(ch)).toBe(Icon.Binary);
  });

  it("should return Icon.Time for a timestamp channel", () => {
    const ch: channel.Payload = {
      key: 1,
      name: "time",
      dataType: DataType.TIMESTAMP,
      isIndex: true,
      index: 0,
      leaseholder: 0,
      virtual: false,
      expression: "",
      internal: false,
      operations: [],
    };
    expect(resolveIcon(ch)).toBe(Icon.Time);
  });

  it("should return Icon.Channel for an unknown data type", () => {
    const ch: channel.Payload = {
      key: 1,
      name: "unknown",
      dataType: DataType.UNKNOWN,
      isIndex: false,
      index: 0,
      leaseholder: 0,
      virtual: false,
      expression: "",
      internal: false,
      operations: [],
    };
    expect(resolveIcon(ch)).toBe(Icon.Channel);
  });
});
