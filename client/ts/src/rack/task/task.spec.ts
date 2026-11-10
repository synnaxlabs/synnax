// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { task as status } from "@/rack/task";
import { task } from "@/task";

describe("rack task", () => {
  it("should load beside the task namespace", () => {
    expect(status.statusConfigZ).toBe(task.keyedConfigZ);
    expect(status.CONFIG_SCHEMAS).toEqual({
      rack_status: { u: status.statusConfigUpdateZ, t: status.statusConfigZ },
    });
  });
});
