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

const MOCK_RESOURCE: ontology.Resource = {
  key: "device:1",
  id: { type: "device", key: "1" },
  name: "test",
};

const mockService = (overrides: Partial<Service>): Service => ({
  ...NOOP_SERVICE,
  type: "device",
  ...overrides,
});

describe("resolveHasChildren", () => {
  it("should return the boolean value when hasChildren is a boolean", () => {
    const service = mockService({ hasChildren: true });
    expect(resolveHasChildren(service, MOCK_RESOURCE)).toBe(true);
  });

  it("should return false when hasChildren is false", () => {
    const service = mockService({ hasChildren: false });
    expect(resolveHasChildren(service, MOCK_RESOURCE)).toBe(false);
  });

  it("should call the function when hasChildren is a function", () => {
    const service = mockService({
      hasChildren: (r: ontology.Resource) => r.name === "test",
    });
    expect(resolveHasChildren(service, MOCK_RESOURCE)).toBe(true);
  });

  it("should return the function result when it returns false", () => {
    const service = mockService({
      hasChildren: (r: ontology.Resource) => r.name === "other",
    });
    expect(resolveHasChildren(service, MOCK_RESOURCE)).toBe(false);
  });
});
