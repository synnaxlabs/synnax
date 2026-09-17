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

describe("EnabledCheckbox", () => {
  it("should render nothing when the field value is absent", async () => {
    await renderInTaskForm(<Task.EnabledCheckbox path="disabled" />, { values: {} });
    expect(screen.queryByRole("checkbox")).toBeNull();
  });

  it("should be checked while the item is enabled", async () => {
    await renderInTaskForm(<Task.EnabledCheckbox path="disabled" />, {
      values: { disabled: false },
    });
    expect(screen.getByRole("checkbox")).toHaveProperty("checked", true);
  });

  it("should keep its name while the item is disabled", async () => {
    await renderInTaskForm(<Task.EnabledCheckbox path="disabled" />, {
      values: { disabled: true },
    });
    expect(screen.getByRole("checkbox", { name: "Enabled" })).toHaveProperty(
      "checked",
      false,
    );
  });

  it("should write the disabled flag when unchecked", async () => {
    const { form } = await renderInTaskForm(<Task.EnabledCheckbox path="disabled" />, {
      values: { disabled: false },
    });
    fireEvent.click(screen.getByRole("checkbox"));
    expect(form.current?.get<boolean>("disabled").value).toBe(true);
  });
});
