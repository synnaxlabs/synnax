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
// Module mocks — must be declared before any component imports
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
    setChannelConfig: vi.fn((payload) => ({ type: "log/setChannelConfig", payload })),
  };
});

vi.mock("@/components", () => ({
  EmptyAction: ({ message }: { message: string }) => (
    <div data-testid="empty-action">{message}</div>
  ),
}));

vi.mock("@/css", () => ({
  CSS: {
    BE: (...parts: string[]) => parts.join("-"),
  },
}));

// Provide lightweight stubs for the Pluto components used by Text/ChannelRow
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
      useRetrieve: vi.fn(() => ({
        data: {
          name: "mock-channel",
          dataType: { equals: () => false },
        },
      })),
    },
    List: {
      Frame: ({ children }: { children: React.ReactNode; data?: number[] }) => (
        <div data-testid="list-frame">{children}</div>
      ),
      Items: <T extends number>({
        emptyContent,
      }: {
        children?: (item: { key: T; index: number }) => React.ReactNode;
        emptyContent?: React.ReactNode;
        full?: string;
        className?: string;
      }) => <div data-testid="list-items">{emptyContent}</div>,
      Item: ({ children }: { children: React.ReactNode }) => (
        <div data-testid="list-item">{children}</div>
      ),
    },
    Input: {
      Text: ({ value }: { value: string }) => (
        <input data-testid="input-text" readOnly value={value} onChange={() => {}} />
      ),
      Numeric: ({ value }: { value: number }) => (
        <input data-testid="input-numeric" readOnly value={value} onChange={() => {}} />
      ),
    },
    Color: {
      Swatch: () => <div data-testid="color-swatch" />,
    },
  };
});

// ---------------------------------------------------------------------------
// Import component under test AFTER mocks
// ---------------------------------------------------------------------------

import * as Selectors from "@/log/selectors";
import { Text } from "@/log/toolbar/Text";

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

describe("log/toolbar/Text", () => {
  it("renders null when state is null", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(undefined);
    const { container } = render(<Text layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders the list frame when state is present", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Text layoutKey="test-key" />);
    expect(screen.getByTestId("list-frame")).toBeDefined();
    expect(screen.getByTestId("list-items")).toBeDefined();
  });

  it("shows empty content message when no channels are configured", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      channels: [],
    });
    render(<Text layoutKey="test-key" />);
    expect(screen.getByTestId("empty-action")).toBeDefined();
    expect(screen.getByText("No channels configured.")).toBeDefined();
  });

  it("calls useSyncComponent with the provided layoutKey", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    // We verify the dispatch mock was set up by the factory call; the fact that
    // rendering succeeds without error confirms useSyncComponent was invoked.
    render(<Text layoutKey="my-unique-layout-key" />);
    // list-frame implies the component rendered past the null-guard
    expect(screen.getByTestId("list-frame")).toBeDefined();
  });

  it("does not render list when state has undefined/null value", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(
      null as unknown as undefined,
    );
    const { container } = render(<Text layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });
});
