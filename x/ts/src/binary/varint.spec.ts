// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { binary } from "@/binary";

describe("encodeVarint", () => {
  it("should encode varints", () => {
    expect(binary.encodeVarint(0)).toEqual(new Uint8Array([0]));
    expect(binary.encodeVarint(1)).toEqual(new Uint8Array([1]));
    expect(binary.encodeVarint(127)).toEqual(new Uint8Array([0x7f]));
    expect(binary.encodeVarint(128)).toEqual(new Uint8Array([0x80, 0x01]));
    expect(binary.encodeVarint(256)).toEqual(new Uint8Array([0x80, 0x02]));
  });
});
describe("decodeVarint", () => {
  it("should decode varints", () => {
    expect(binary.decodeVarint(new Uint8Array([0x80, 0x01]))).toEqual([128, 2]);
    expect(binary.decodeVarint(new Uint8Array([0x01]))).toEqual([1, 1]);
    expect(binary.decodeVarint(new Uint8Array([0x00]))).toEqual([0, 1]);
  });
  it("should throw if the varint is not found", () => {
    expect(() => binary.decodeVarint(new Uint8Array([0x80]))).toThrow();
  });
});
