// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { NOOP_SERVICE, resolveHasChildren, type Service } from "@/ontology/service";

const mockResource = (data?: Record<string, unknown>): ontology.Resource =>
  ({
    id: { type: "test", key: "1" },
    name: "test",
    data,
  }) as unknown as ontology.Resource;

describe("resolveHasChildren", () => {
  it("should return the boolean value when hasChildren is a boolean", () => {
    const service = {
      ...NOOP_SERVICE,
      type: "test",
      hasChildren: true,
    } as unknown as Service;
    expect(resolveHasChildren(service, mockResource())).toBe(true);
  });

  it("should return false when hasChildren is false", () => {
    const service = {
      ...NOOP_SERVICE,
      type: "test",
      hasChildren: false,
    } as unknown as Service;
    expect(resolveHasChildren(service, mockResource())).toBe(false);
  });

  it("should call the function when hasChildren is a function", () => {
    const service = {
      ...NOOP_SERVICE,
      type: "test",
      hasChildren: (r: ontology.Resource) => r.name === "test",
    } as unknown as Service;
    expect(resolveHasChildren(service, mockResource())).toBe(true);
  });

  it("should return the function result when it returns false", () => {
    const service = {
      ...NOOP_SERVICE,
      type: "test",
      hasChildren: (r: ontology.Resource) => r.name === "other",
    } as unknown as Service;
    expect(resolveHasChildren(service, mockResource())).toBe(false);
  });
});
