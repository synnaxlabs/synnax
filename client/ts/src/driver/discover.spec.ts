// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { driver } from "@/driver";

describe("payloadZ", () => {
  it("should decode a base64 payload into its bytes", () => {
    const bytes = driver.payloadZ.parse(btoa('{"v":1}'));
    expect(new TextDecoder().decode(bytes)).toEqual('{"v":1}');
  });

  it("should decode an empty payload into no bytes", () => {
    expect(driver.payloadZ.parse("")).toHaveLength(0);
  });
});
