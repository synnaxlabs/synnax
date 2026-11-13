// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ni } from "@/ni";

describe("Update", () => {
  it("should parse an identity update by its group", () => {
    const update = ni.device.updateZ.parse({
      group: "identity",
      rack: 1,
      serial: "01A",
      model: "PXI-6255",
      alias: "Mod1",
    });
    expect(update.group).toEqual("identity");
    if (update.group !== "identity") return;
    expect(update.serial).toEqual("01A");
  });

  it("should parse a configuration update by its group", () => {
    const update = ni.device.updateZ.parse({ group: "configuration", name: "Module" });
    expect(update.group).toEqual("configuration");
    if (update.group !== "configuration") return;
    expect(update.name).toEqual("Module");
  });

  it("should reject an unknown group", () => {
    expect(
      ni.device.updateZ.safeParse({ group: "other", name: "Module" }).success,
    ).toBe(false);
  });
});
