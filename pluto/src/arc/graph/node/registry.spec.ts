// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Node } from "@/arc/graph/node";

describe("Arc.Node.resolveSpec", () => {
  it("should return the registered spec of a function type", () => {
    expect(Node.resolveSpec("on")).toBe(Node.REGISTRY.on);
  });

  it("should throw a not found error for an unknown function type", () => {
    expect(() => Node.resolveSpec("not-a-function")).toThrow(NotFoundError);
  });
});
