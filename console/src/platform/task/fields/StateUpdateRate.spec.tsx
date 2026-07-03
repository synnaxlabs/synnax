// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  commitFieldInput,
  findFieldInput,
  renderInTaskForm,
} from "@/platform/task/testutil";

describe("fields.StateUpdateRate", () => {
  it("should seed the input from config.stateRate and write changes back", async () => {
    const { form } = await renderInTaskForm(<Task.Fields.StateUpdateRate />, {
      values: { config: { stateRate: 10 } },
    });
    const input = findFieldInput();
    expect(input.value).toBe("10");
    commitFieldInput(input, "20");
    expect(form.current?.get("config.stateRate").value).toBe(20);
  });
});
