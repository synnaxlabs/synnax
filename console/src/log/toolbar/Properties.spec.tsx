// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";

import { type State } from "@/log/types/v0";

const mockDispatch = vi.fn();

vi.mock("@/log/Log", () => ({
  useSyncComponent: () => mockDispatch,
}));

vi.mock("@/log/selectors", () => ({
  useSelectOptional: vi.fn(),
}));

vi.mock("@/log/slice", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    setTimestampPrecision: vi.fn((payload) => ({
      type: "log/setTimestampPrecision",
      payload,
    })),
    setShowChannelNames: vi.fn((payload) => ({
      type: "log/setShowChannelNames",
      payload,
    })),
  };
});

vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    Access: {
      ...((actual as Record<string, unknown>).Access as object),
      useUpdateGranted: vi.fn(() => true),
    },
    Flex: {
      Box: ({
        children,
        style,
      }: {
        children: React.ReactNode;
        x?: boolean;
        style?: React.CSSProperties;
      }) => (
        <div data-testid="flex-box" style={style}>
          {children}
        </div>
      ),
    },
    Input: {
      Item: ({ children, label }: { children: React.ReactNode; label: string }) => (
        <label data-testid={`input-item-${label.toLowerCase().replace(/ /g, "-")}`}>
          {label}
          {children}
        </label>
      ),
      Numeric: ({
        value,
        disabled,
      }: {
        value: number;
        onChange: (v: number) => void;
        disabled: boolean;
        resetValue?: number;
        bounds?: object;
      }) => (
        <input
          data-testid="input-numeric"
          readOnly
          value={value}
          data-disabled={String(disabled)}
          onChange={() => {}}
        />
      ),
      Switch: ({
        value,
        disabled,
        onChange,
      }: {
        value: boolean;
        onChange: (v: boolean) => void;
        disabled: boolean;
      }) => (
        <button
          data-testid="input-switch"
          data-checked={String(value)}
          data-disabled={String(disabled)}
          onClick={() => onChange(!value)}
        />
      ),
    },
  };
});

import * as Pluto from "@synnaxlabs/pluto";

import * as Selectors from "@/log/selectors";
import * as Slice from "@/log/slice";
import { Properties } from "@/log/toolbar/Properties";

const ZERO_STATE: State = {
  key: "test-key",
  version: "0.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  channelConfigs: {},
  showChannelNames: true,
};

describe("log/toolbar/Properties", () => {
  it("renders null when state is null", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(undefined);
    const { container } = render(<Properties layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders timestamp precision and show channel names controls", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Properties layoutKey="test-key" />);
    expect(screen.getByText("Timestamp Precision")).toBeDefined();
    expect(screen.getByText("Show Channel Names")).toBeDefined();
  });

  it("renders the current timestamp precision value", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      timestampPrecision: 2,
    });
    render(<Properties layoutKey="test-key" />);
    const input = screen.getByTestId("input-numeric");
    expect(Number((input as HTMLInputElement).value)).toBe(2);
  });

  it("renders the current showChannelNames value", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      showChannelNames: false,
    });
    render(<Properties layoutKey="test-key" />);
    expect(screen.getByTestId("input-switch").getAttribute("data-checked")).toBe(
      "false",
    );
  });

  it("dispatches setShowChannelNames when toggle is clicked", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Properties layoutKey="test-key" />);
    fireEvent.click(screen.getByTestId("input-switch"));
    expect(mockDispatch).toHaveBeenCalled();
    expect(Slice.setShowChannelNames).toHaveBeenCalledWith({
      key: "test-key",
      showChannelNames: false,
    });
  });

  it("disables controls when user lacks edit permission", () => {
    vi.mocked(Pluto.Access.useUpdateGranted).mockReturnValueOnce(false);
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Properties layoutKey="test-key" />);
    expect(screen.getByTestId("input-numeric").getAttribute("data-disabled")).toBe(
      "true",
    );
    expect(screen.getByTestId("input-switch").getAttribute("data-disabled")).toBe(
      "true",
    );
  });
});
