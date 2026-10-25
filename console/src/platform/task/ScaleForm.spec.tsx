// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack } from "@synnaxlabs/client";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

const renderScale = (scale: labjack.Scale) =>
  renderInTaskForm(<Task.ScaleForm path="scale" schemas={labjack.SCALE_SCHEMAS} />, {
    values: { scale },
  });

const valueOf = (name: string): string =>
  screen.getByRole<HTMLInputElement>("textbox", { name }).value;

describe("ScaleForm", () => {
  it("should show the slope and offset of a linear scale", async () => {
    await renderScale({ type: "linear", slope: 2, offset: 1 });
    expect(valueOf("Slope")).toBe("2");
    expect(valueOf("Offset")).toBe("1");
  });

  it("should show the ranges of a map scale", async () => {
    await renderScale({
      type: "map",
      preScaledMin: 4,
      preScaledMax: 20,
      scaledMin: 1,
      scaledMax: 100,
    });
    expect(valueOf("Pre-scaled min")).toBe("4");
    expect(valueOf("Pre-scaled max")).toBe("20");
    expect(valueOf("Scaled min")).toBe("1");
    expect(valueOf("Scaled max")).toBe("100");
  });

  it("should show no parameters when the channel is not scaled", async () => {
    await renderScale({ type: "none" });
    expect(screen.queryByText("Slope")).toBeNull();
    expect(screen.queryByText("Pre-scaled min")).toBeNull();
  });

  it("should fill the defaults of the type picked", async () => {
    const { form } = await renderScale({ type: "none" });
    fireEvent.click(screen.getByRole("button", { name: /None/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Linear/ }));
    expect(form.current?.get<labjack.Scale>("scale").value).toEqual({
      type: "linear",
      slope: 1,
      offset: 0,
    });
    expect(await screen.findByText("Slope")).toBeTruthy();
  });

  it("should drop the parameters of the previous type", async () => {
    const { form } = await renderScale({ type: "linear", slope: 2, offset: 1 });
    fireEvent.click(screen.getByRole("button", { name: /Linear/ }));
    fireEvent.click(await screen.findByRole("option", { name: /None/ }));
    expect(form.current?.get<labjack.Scale>("scale").value).toEqual({ type: "none" });
  });
});
