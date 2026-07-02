// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/platform/range";

const services: Range.SnapshotServices = {
  schematic: {
    icon: <Icon.Snapshot />,
    onClick: vi.fn(async () => {}),
    onDelete: vi.fn(async () => {}),
  },
};

const Consumer = (): ReactElement => {
  const resolved = Range.useSnapshotServices();
  return <span>{Object.keys(resolved).join(",")}</span>;
};

describe("Range snapshots", () => {
  it("should expose the provided services to consumers below the provider", () => {
    render(
      <Range.SnapshotServicesProvider services={services}>
        <Consumer />
      </Range.SnapshotServicesProvider>,
    );
    expect(screen.getByText("schematic")).toBeTruthy();
  });

  it("should throw when the hook is used outside the provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => render(<Consumer />)).toThrow(
      /must be used within Range.SnapshotServicesProvider/,
    );
    spy.mockRestore();
  });
});
