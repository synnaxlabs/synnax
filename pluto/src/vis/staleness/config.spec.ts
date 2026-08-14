// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Staleness } from "@/vis/staleness";

describe("Staleness.configZ", () => {
  // A symbol saved before it gained staleness config carries neither key.
  it("should accept a config that carries neither key", () => {
    expect(Staleness.configZ.parse({})).toEqual({});
  });

  it("should accept the config a symbol starts with", () => {
    expect(Staleness.configZ.parse(Staleness.ZERO_CONFIG)).toEqual(
      Staleness.ZERO_CONFIG,
    );
  });

  // The symbol schema and the aether schema must agree, so the form rejects a timeout
  // the worker would rather than pushing it across and failing there.
  it.each([0, -1])("should reject a non-positive timeout of %s", (t) => {
    expect(() => Staleness.configZ.parse({ stalenessTimeout: t })).toThrow();
  });
});
