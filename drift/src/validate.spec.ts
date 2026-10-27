// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type StoreState } from "@/index";
import { configureMockStore, MockRuntime } from "@/testutil";

describe("validate", () => {
  it("should reject a dispatched action with an empty type", async () => {
    const store = await configureMockStore();
    expect(() => store.dispatch({ type: "" })).toThrow();
  });

  it("should reject an emitted action with an empty type", async () => {
    const runtime = new MockRuntime<StoreState>(true, { key: "main" });
    await configureMockStore(runtime);
    expect(() =>
      runtime.subscribeCallback({ action: { type: "" }, emitter: "b" }),
    ).toThrow();
  });

  it("should accept an action with a type", async () => {
    const store = await configureMockStore();
    expect(() => store.dispatch({ type: "type" })).not.toThrow();
  });
});
