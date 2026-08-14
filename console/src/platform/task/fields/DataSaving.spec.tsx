// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

describe("fields.DataSaving", () => {
  it("should write the toggled value back into the form", async () => {
    const { form } = await renderInTaskForm(<Task.Fields.DataSaving />, {
      values: { config: { dataSaving: true } },
    });
    const checkbox = document.body.querySelector<HTMLInputElement>(
      "input[type='checkbox']",
    );
    expect(checkbox).toBeTruthy();
    fireEvent.click(checkbox as HTMLInputElement);
    expect(form.current?.get("config.dataSaving").value).toBe(false);
  });
});
