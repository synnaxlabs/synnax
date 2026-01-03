// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { createMockWorkers } from "@/worker/worker";

interface MockMessage {
  value: string;
}

describe("Worker", () => {
  it("should correctly send and handle a message from a typed worker", () => {
    const [a, b] = createMockWorkers();
    const aT = a.route<MockMessage>("dog");
    const bT = b.route<MockMessage>("dog");
    const handler = vi.fn();
    bT.handle(handler);
    aT.send({ value: "hello" });
    expect(handler).toHaveBeenCalledWith({ value: "hello" });
  });
  it("should route the message to the correct location", () => {
    const [a, b] = createMockWorkers();
    const aT = a.route<MockMessage>("dog");
    const bT = b.route<MockMessage>("cat");
    const dogBT = b.route<MockMessage>("dog");
    const handler = vi.fn();
    bT.handle(handler);
    const dogHandler = vi.fn();
    dogBT.handle(dogHandler);
    aT.send({ value: "hello" });
    expect(handler).not.toHaveBeenCalled();
    expect(dogHandler).toHaveBeenCalled();
  });
});
