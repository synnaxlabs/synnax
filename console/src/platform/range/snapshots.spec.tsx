// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/platform/range";

const services: Range.SnapshotServices = {
  schematic: {
    icon: <Icon.Snapshot />,
    useIsSnapshot: () => true,
    onClick: vi.fn(async () => {}),
    onDelete: vi.fn(async () => {}),
  },
};

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <Range.SnapshotServicesProvider services={services}>
    {children}
  </Range.SnapshotServicesProvider>
);

describe("Range snapshots", () => {
  it("should expose the provided services to consumers below the provider", () => {
    const { result } = renderHook(() => Range.useSnapshotServices(), { wrapper });
    expect(result.current).toBe(services);
  });

  it("should throw when the hook is used outside the provider", () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    expect(() => renderHook(() => Range.useSnapshotServices())).toThrow(
      /must be used within Range.SnapshotServicesProvider/,
    );
    spy.mockRestore();
  });
});
