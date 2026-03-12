// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";

import { type State } from "@/log/types/v0";

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

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
    setChannels: vi.fn((payload) => ({ type: "log/setChannels", payload })),
    setTimestampPrecision: vi.fn((payload) => ({
      type: "log/setTimestampPrecision",
      payload,
    })),
  };
});

// Stub Pluto primitives that Channels.tsx uses
vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    Access: {
      ...((actual as Record<string, unknown>).Access as object),
      useUpdateGranted: vi.fn(() => true),
    },
    Channel: {
      ...((actual as Record<string, unknown>).Channel as object),
      SelectMultiple: ({
        value,
        disabled,
      }: {
        value: number[];
        onChange: (v: number[]) => void;
        disabled: boolean;
        initialQuery?: object;
      }) => (
        <div
          data-testid="channel-select-multiple"
          data-disabled={String(disabled)}
          data-value={JSON.stringify(value)}
        />
      ),
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
      Item: ({
        children,
        label,
      }: {
        children: React.ReactNode;
        label: string;
        grow?: boolean;
      }) => (
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
    },
  };
});

// ---------------------------------------------------------------------------
// Import component under test
// ---------------------------------------------------------------------------

import * as Pluto from "@synnaxlabs/pluto";

import * as Selectors from "@/log/selectors";
import { Channels } from "@/log/toolbar/Channels";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const ZERO_STATE: State = {
  key: "test-key",
  version: "0.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  channelConfigs: {},
};

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("log/toolbar/Channels", () => {
  it("renders null when state is null", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(undefined);
    const { container } = render(<Channels layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders null when state is null-ish", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(
      null as unknown as undefined,
    );
    const { container } = render(<Channels layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders the channel selector and precision input when state is present", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByTestId("channel-select-multiple")).toBeDefined();
    expect(screen.getByTestId("input-numeric")).toBeDefined();
  });

  it("renders Channels label", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByText("Channels")).toBeDefined();
  });

  it("renders Timestamp Precision label", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByText("Timestamp Precision")).toBeDefined();
  });

  it("passes current channels from state to the selector", () => {
    const stateWithChannels: State = { ...ZERO_STATE, channels: [10, 20, 30] };
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(stateWithChannels);
    render(<Channels layoutKey="test-key" />);
    const selector = screen.getByTestId("channel-select-multiple");
    expect(selector.getAttribute("data-value")).toBe(JSON.stringify([10, 20, 30]));
  });

  it("passes current timestampPrecision to the numeric input", () => {
    const stateWithPrecision: State = { ...ZERO_STATE, timestampPrecision: 3 };
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(stateWithPrecision);
    render(<Channels layoutKey="test-key" />);
    const numericInput = screen.getByTestId("input-numeric");
    expect(Number((numericInput as HTMLInputElement).value)).toBe(3);
  });

  it("disables inputs when user lacks edit permission", () => {
    vi.mocked(Pluto.Access.useUpdateGranted).mockReturnValueOnce(false);
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Channels layoutKey="test-key" />);
    const selector = screen.getByTestId("channel-select-multiple");
    expect(selector.getAttribute("data-disabled")).toBe("true");
    const numericInput = screen.getByTestId("input-numeric");
    expect(numericInput.getAttribute("data-disabled")).toBe("true");
  });

  it("enables inputs when user has edit permission", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Channels layoutKey="test-key" />);
    const selector = screen.getByTestId("channel-select-multiple");
    expect(selector.getAttribute("data-disabled")).toBe("false");
    const numericInput = screen.getByTestId("input-numeric");
    expect(numericInput.getAttribute("data-disabled")).toBe("false");
  });
});
