// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type group } from "@/group";
import { project } from "@/project";

describe("builder", () => {
  it("should reject a condition of an entity with the same shape", () => {
    const key = "00000000-0000-4000-8000-000000000001";
    // @ts-expect-error a project condition is not a group condition
    const condition: group.Condition = project.key.eq(key);
    expect(condition.tree).toEqual({ key: { eq: key } });
  });
});
