// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

describe("fields.AutoStart", () => {
  it("should render the Auto start switch when the field is present", async () => {
    await renderInTaskForm(<Task.Fields.AutoStart />, {
      values: { config: { autoStart: false } },
    });
    expect(screen.getByText("Auto start")).toBeTruthy();
  });

  it("should render nothing when the field is null (hideIfNull)", async () => {
    await renderInTaskForm(<Task.Fields.AutoStart />, { values: { config: {} } });
    expect(screen.queryByText("Auto start")).toBeNull();
  });

  it("should write the toggled value back into the form", async () => {
    const { form } = await renderInTaskForm(<Task.Fields.AutoStart />, {
      values: { config: { autoStart: false } },
    });
    const checkbox = document.body.querySelector<HTMLInputElement>(
      "input[type='checkbox']",
    );
    expect(checkbox).toBeTruthy();
    fireEvent.click(checkbox as HTMLInputElement);
    expect(form.current?.get("config.autoStart").value).toBe(true);
  });
});
