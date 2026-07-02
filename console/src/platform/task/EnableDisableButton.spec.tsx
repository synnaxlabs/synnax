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

describe("EnableDisableButton", () => {
  it("should render a toggle when the field is present", async () => {
    await renderInTaskForm(<Task.EnableDisableButton path="enabled" />, {
      values: { enabled: true },
    });
    expect(screen.getByRole("button")).toBeTruthy();
  });

  it("should render nothing when the field value is absent", async () => {
    await renderInTaskForm(<Task.EnableDisableButton path="enabled" />, {
      values: {},
    });
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("should write the toggled value back into the form", async () => {
    const { form } = await renderInTaskForm(
      <Task.EnableDisableButton path="enabled" />,
      { values: { enabled: true } },
    );
    fireEvent.click(screen.getByRole("button"));
    expect(form.current?.get("enabled").value).toBe(false);
  });

  it("should be disabled when the task is a snapshot", async () => {
    const { form } = await renderInTaskForm(
      <Task.EnableDisableButton path="enabled" />,
      { values: { enabled: true, snapshot: true } },
    );
    fireEvent.click(screen.getByRole("button"));
    expect(form.current?.get("enabled").value).toBe(true);
  });
});
