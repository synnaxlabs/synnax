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

import { ingestComponent } from "./ingestComponent";
import { type FileIngesterContext, type FileIngesters } from "./ingester";

const stubContext = (): FileIngesterContext => ({
  layout: { name: "test" },
  placeLayout: vi.fn(),
  store: {} as FileIngesterContext["store"],
  client: null,
});

describe("ingestComponent", () => {
  it("should use the typed ingester when data has a matching type field", () => {
    const ingester = vi.fn();
    const ingesters: FileIngesters = { lineplot: ingester };
    const data = { type: "lineplot", key: "lp-1" };
    const ctx = stubContext();

    ingestComponent(data, "test.json", ingesters, ctx);

    expect(ingester).toHaveBeenCalledOnce();
    expect(ingester).toHaveBeenCalledWith(data, ctx);
  });

  it("should fall back to trying all ingesters when data has no type field", () => {
    const failIngester = vi.fn().mockImplementation(() => {
      throw new ZodError([]);
    });
    const successIngester = vi.fn();
    const ingesters: FileIngesters = {
      schematic: failIngester,
      lineplot: successIngester,
    };
    const data = { key: "lp-1" };
    const ctx = stubContext();

    ingestComponent(data, "test.json", ingesters, ctx);

    expect(failIngester).toHaveBeenCalledOnce();
    expect(successIngester).toHaveBeenCalledOnce();
  });

  it("should stop trying ingesters after the first success", () => {
    const first = vi.fn();
    const second = vi.fn();
    const ingesters: FileIngesters = { a: first, b: second };

    ingestComponent({ key: "x" }, "test.json", ingesters, stubContext());

    expect(first).toHaveBeenCalledOnce();
    expect(second).not.toHaveBeenCalled();
  });

  it("should throw when no ingester can handle the data", () => {
    const ingester = vi.fn().mockImplementation(() => {
      throw new ZodError([]);
    });
    const ingesters: FileIngesters = { schematic: ingester };

    expect(() =>
      ingestComponent({ key: "unknown" }, "mystery.json", ingesters, stubContext()),
    ).toThrow("mystery.json cannot be imported");
  });

  it("should fall back to trying all ingesters when type does not match any registered ingester", () => {
    const ingester = vi.fn();
    const ingesters: FileIngesters = { lineplot: ingester };
    const data = { type: "unknown_type", key: "x" };
    const ctx = stubContext();

    ingestComponent(data, "test.json", ingesters, ctx);

    expect(ingester).toHaveBeenCalledOnce();
    expect(ingester).toHaveBeenCalledWith(data, ctx);
  });

  it("should re-throw non-Zod errors immediately", () => {
    const ingester = vi.fn().mockImplementation(() => {
      throw new TypeError("something broke");
    });
    const ingesters: FileIngesters = { schematic: ingester };

    expect(() =>
      ingestComponent({ key: "x" }, "test.json", ingesters, stubContext()),
    ).toThrow("something broke");
  });
});
