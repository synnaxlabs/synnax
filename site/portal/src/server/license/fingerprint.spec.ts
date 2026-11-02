// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { parse, read } from "@/server/license/fingerprint";
import { HASH_A, HASH_B } from "@/server/license/testutil";

describe("fingerprint.parse", () => {
  it("should split on commas, spaces, and new lines and drop duplicates", () => {
    expect(parse(`${HASH_A}, ${HASH_B}\n${HASH_A}  `)).toEqual([HASH_A, HASH_B]);
  });

  it("should lowercase uppercase hex", () => {
    expect(parse(HASH_A.toUpperCase())).toEqual([HASH_A]);
  });

  it("should reject an empty paste", () => {
    expect(() => parse(" \n ")).toThrow("Enter the machine's fingerprint");
  });

  it("should name the value that is not a hash", () => {
    expect(() => parse(`${HASH_A}, nope`)).toThrow(
      '"nope" is not part of a fingerprint',
    );
  });
});

describe("fingerprint.read", () => {
  it("should read a fingerprint pasted on its own", () => {
    expect(read(` ${HASH_A}\n${HASH_B} `)).toEqual([HASH_A, HASH_B]);
  });

  it("should read the fingerprint inside an activation link", () => {
    const link = `https://portal.synnaxlabs.com/licenses/activate?fingerprint=${HASH_A}%2C${HASH_B}&name=rig`;
    expect(read(link)).toEqual([HASH_A, HASH_B]);
  });

  it("should reject a link that carries no fingerprint", () => {
    expect(() => read("https://portal.synnaxlabs.com/licenses/activate")).toThrow(
      "Enter the machine's fingerprint",
    );
  });
});
