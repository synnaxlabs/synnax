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

describe("fields.StreamRate", () => {
  it("should seed the input from config.streamRate and write changes back", async () => {
    const { form } = await renderInTaskForm(<Task.Fields.StreamRate />, {
      values: { config: { streamRate: 5 } },
    });
    const input = findFieldInput();
    expect(input.value).toBe("5");
    commitFieldInput(input, "8");
    expect(form.current?.get("config.streamRate").value).toBe(8);
  });
});
