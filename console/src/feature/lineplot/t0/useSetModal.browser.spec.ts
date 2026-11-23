// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { T0 } from "@/feature/lineplot/t0";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const RANGE = new TimeRange(TimeStamp.seconds(100), TimeStamp.seconds(200));
const VALUE = Number(RANGE.start.add(TimeSpan.seconds(5)).valueOf());

const openSetT0 = async () => {
  const handle = await renderModalOpener(T0.useSetModal, [
    { name: "Test 1", timeRange: RANGE, value: VALUE },
  ]);
  await screen.findByText("Set T0 of Test 1");
  return handle;
};

describe("T0.useSetModal", () => {
  it("should return the T0 on save", async () => {
    const { result } = await openSetT0();
    fireEvent.click(findButton("Save"));
    await expect(result()).resolves.toBe(VALUE);
  });

  it("should return nothing on cancel", async () => {
    const { result } = await openSetT0();
    fireEvent.click(findButton("Cancel"));
    await expect(result()).resolves.toBeNull();
  });
});
