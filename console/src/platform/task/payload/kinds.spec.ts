// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type payload } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";

describe("channelKinds", () => {
  it.each<[DataType, payload.Kind[]]>([
    [DataType.FLOAT32, ["float", "string", "boolean"]],
    [DataType.INT64, ["integer", "string", "boolean"]],
    [DataType.UINT8, ["unsigned", "string", "boolean"]],
    [DataType.BOOLEAN, ["boolean", "unsigned", "string"]],
    [DataType.STRING, ["string"]],
    [DataType.TIMESTAMP, ["timestamp"]],
    [DataType.JSON, ["json"]],
    [DataType.UUID, ["uuid"]],
    [DataType.BYTES, ["bytes"]],
  ])("should list the kinds a %s channel can be written as", (dt, kinds) => {
    expect(Task.Payload.channelKinds(dt)).toEqual(kinds);
  });
});

describe("newChannelDataType", () => {
  it.each<[payload.Kind, string]>([
    ["float", "float64"],
    ["integer", "int64"],
    ["unsigned", "uint64"],
    ["string", "string"],
    ["boolean", "uint8"],
    ["timestamp", "timestamp"],
    ["json", "json"],
    ["uuid", "uuid"],
    ["bytes", "bytes"],
  ])("should give a new %s channel the %s data type", (kind, dt) => {
    expect(Task.Payload.newChannelDataType(kind)).toBe(dt);
  });
});
