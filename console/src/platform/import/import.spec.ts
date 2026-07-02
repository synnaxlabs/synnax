// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";
import { ZodError } from "zod";

import { Import } from "@/platform/import";

const ctx = {
  layout: { name: "test" },
  placeLayout: vi.fn(),
  store: {} as never,
  client: null,
  projectKey: "project-1",
} as unknown as Import.FileIngesterContext;

describe("ingestComponent", () => {
  it("dispatches typed data to the ingester matching its type", async () => {
    const log = vi.fn();
    const table = vi.fn();
    const data = { type: "log", key: "abc" };
    await Import.ingestComponent(data, "log.json", { log, table }, ctx);
    expect(log).toHaveBeenCalledTimes(1);
    expect(log).toHaveBeenCalledWith(data, ctx);
    expect(table).not.toHaveBeenCalled();
  });

  it("tries each ingester in turn for untyped data, stopping at the first success", async () => {
    const first = vi.fn().mockRejectedValue(new ZodError([]));
    const second = vi.fn().mockResolvedValue(undefined);
    const third = vi.fn();
    const data = { key: "no-type-field" };
    await Import.ingestComponent(data, "unknown.json", { first, second, third }, ctx);
    expect(first).toHaveBeenCalledWith(data, ctx);
    expect(second).toHaveBeenCalledWith(data, ctx);
    expect(third).not.toHaveBeenCalled();
  });

  it("throws a cannot-be-imported error when every ingester rejects with a ZodError", async () => {
    const first = vi.fn().mockRejectedValue(new ZodError([]));
    const second = vi.fn().mockRejectedValue(new ZodError([]));
    await expect(
      Import.ingestComponent({ key: "x" }, "bad.json", { first, second }, ctx),
    ).rejects.toThrow("bad.json cannot be imported.");
  });

  it("rethrows a non-Zod error raised by an untyped ingester without trying the rest", async () => {
    const boom = new Error("disk on fire");
    const first = vi.fn().mockRejectedValue(boom);
    const second = vi.fn();
    await expect(
      Import.ingestComponent({ key: "x" }, "boom.json", { first, second }, ctx),
    ).rejects.toThrow("disk on fire");
    expect(second).not.toHaveBeenCalled();
  });
});
