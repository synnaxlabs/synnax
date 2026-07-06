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

describe("fields.SampleRate", () => {
  it("should seed the input from config.sampleRate and write changes back", async () => {
    const { form } = await renderInTaskForm(<Task.Fields.SampleRate />, {
      values: { config: { sampleRate: 25 } },
    });
    const input = findFieldInput();
    expect(input.value).toBe("25");
    commitFieldInput(input, "50");
    expect(form.current?.get("config.sampleRate").value).toBe(50);
  });
});
