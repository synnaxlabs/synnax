// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { bus } from "@/bus";

describe("framingZ", () => {
  it("should read and write a length size as its schema value", () => {
    const framing = binary.JSON_CODEC.decodeString(
      '{"type": "sync", "sync": "qg==", "length_size": 4}',
      bus.framingZ,
    );
    expect(framing).toMatchObject({ type: "sync", lengthSize: bus.LengthSize.four });
    expect(binary.JSON_CODEC.encodeString(framing, bus.framingZ)).toContain(
      '"length_size":4',
    );
  });

  it.each([
    ["an empty sync sequence", { type: "sync", sync: "" }, ["sync"]],
    [
      "a length size of 3",
      { type: "sync", sync: "qg==", lengthSize: 3 },
      ["lengthSize"],
    ],
    ["a max length of 0", { type: "sync", sync: "qg==", maxLength: 0 }, ["maxLength"]],
    [
      "an unknown checksum algorithm",
      { type: "sync", sync: "qg==", checksum: { algorithm: "crc8" } },
      ["checksum", "algorithm"],
    ],
    ["a fixed length of 0", { type: "fixed", length: 0 }, ["length"]],
    ["a fixed length of 65537", { type: "fixed", length: 65537 }, ["length"]],
    ["an empty delimiter", { type: "delimited", delimiter: "" }, ["delimiter"]],
  ])("should refuse %s at its field", (_, value, field) => {
    const { error } = bus.framingZ.safeParse(value);
    expect(error?.issues.map(({ path }) => path)).toEqual([field]);
  });
});

describe("pollZ", () => {
  it.each([
    ["an empty query", { query: "", rate: 1 }, ["query"]],
    ["a rate of 0", { query: "qg==", rate: 0 }, ["rate"]],
  ])("should refuse %s at its field", (_, value, field) => {
    const { error } = bus.pollZ.safeParse(value);
    expect(error?.issues.map(({ path }) => path)).toEqual([field]);
  });
});
