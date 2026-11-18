// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Ranger } from "@/ranger";

const START = TimeStamp.now();

const createRange = (fields: Partial<ranger.New>): ranger.Payload =>
  ranger.payloadZ.parse({
    key: "7a2b2f2e-6b3c-4b8e-9c55-1f0b6a3d4e5f",
    name: "Hotfire 12",
    timeRange: { start: START, end: START.add(TimeSpan.hours(2)) },
    ...fields,
  });

describe("Ranger.HaulPreview", () => {
  it("should show the start and the length of the range", () => {
    const { container } = render(<Ranger.HaulPreview range={createRange({})} />);
    expect(container.querySelector(".lyra-haul-preview__name")?.textContent).toBe(
      "Hotfire 12",
    );
    expect(container.querySelector(".lyra-haul-preview__facts")?.textContent).toBe(
      `${START.toString("dateTime", "local")}·2h`,
    );
  });

  it("should show the start alone for a range with no end", () => {
    const open = createRange({ timeRange: { start: START, end: TimeStamp.MAX } });
    const { container } = render(<Ranger.HaulPreview range={open} />);
    expect(container.querySelector(".lyra-haul-preview__facts")?.textContent).toBe(
      START.toString("dateTime", "local"),
    );
  });

  it("should call a range with no start planned", () => {
    const planned = createRange({
      timeRange: { start: TimeStamp.MAX, end: TimeStamp.MAX },
    });
    const { container } = render(<Ranger.HaulPreview range={planned} />);
    expect(container.querySelector(".lyra-haul-preview__facts")?.textContent).toBe(
      "Planned",
    );
  });

  it("should fill the tile with the color of the range", () => {
    const { container } = render(
      <Ranger.HaulPreview range={createRange({ color: "#00ff00" })} />,
    );
    const tile = container.querySelector<HTMLElement>(".lyra-haul-preview__tile");
    expect(tile?.style.background).toBe("rgb(0, 255, 0)");
  });
});
