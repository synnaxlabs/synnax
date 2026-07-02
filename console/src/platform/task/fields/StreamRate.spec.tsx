// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

describe("fields.StreamRate", () => {
  it("should render the Stream Rate label with a Hz unit", async () => {
    await renderInTaskForm(<Task.Fields.StreamRate />, {
      values: { config: { streamRate: 5 } },
    });
    expect(screen.getByText("Stream Rate")).toBeTruthy();
    expect(screen.getByText("Hz")).toBeTruthy();
  });

  it("should seed the input from the form value", async () => {
    await renderInTaskForm(<Task.Fields.StreamRate />, {
      values: { config: { streamRate: 5 } },
    });
    const input = document.body.querySelector<HTMLInputElement>("input");
    expect(input?.value).toBe("5");
  });
});
