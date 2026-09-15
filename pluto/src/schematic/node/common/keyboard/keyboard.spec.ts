// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Keyboard } from "@/schematic/node/common/keyboard";

const event = (key: string) => {
  let prevented = false;
  return {
    e: { key, preventDefault: () => (prevented = true) } as never,
    prevented: () => prevented,
  };
};

describe("Keyboard.blockActivation", () => {
  it.each([" ", "Enter"])("should prevent the default for %j", (key) => {
    const { e, prevented } = event(key);
    Keyboard.blockActivation(e);
    expect(prevented()).toBe(true);
  });

  it("should leave other keys untouched", () => {
    const { e, prevented } = event("a");
    Keyboard.blockActivation(e);
    expect(prevented()).toBe(false);
  });
});
