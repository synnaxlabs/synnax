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

import { type State } from "@/log/types/v1";

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
    addChannel: vi.fn((payload) => ({ type: "log/addChannel", payload })),
    removeChannelByIndex: vi.fn((payload) => ({
      type: "log/removeChannelByIndex",
      payload,
    })),
    setChannelAtIndex: vi.fn((payload) => ({
      type: "log/setChannelAtIndex",
      payload,
    })),
    setChannelConfig: vi.fn((payload) => ({
      type: "log/setChannelConfig",
      payload,
    })),
  };
});

vi.mock("@/css", () => ({
  CSS: {
    BE: (...parts: string[]) => parts.join("-"),
  },
}));

vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as object),
    Access: {
      ...((actual as Record<string, unknown>).Access as object),
      useUpdateGranted: vi.fn(() => true),
    },
    Theming: {
      use: () => ({
        colors: {
          gray: { l11: "#888888" },
          primary: { z: "#0000ff" },
        },
      }),
    },
    Notation: {
      Select: ({
        value,
        disabled,
      }: {
        value: string | undefined;
        onChange: (v: string) => void;
        allowNone?: boolean;
        disabled?: boolean;
      }) => (
        <div
          data-testid="notation-select"
          data-value={value ?? ""}
          data-disabled={String(disabled ?? false)}
        />
      ),
    },
    Channel: {
      ...((actual as Record<string, unknown>).Channel as object),
      SelectSingle: ({
        value,
        disabled,
        triggerProps,
      }: {
        value: number;
        onChange: (v: number) => void;
        disabled: boolean;
        initialQuery?: object;
        grow?: boolean;
        variant?: string;
        triggerProps?: { placeholder?: string };
      }) => (
        <div
          data-testid={
            triggerProps?.placeholder ? "add-channel-select" : `channel-select-${value}`
          }
          data-disabled={String(disabled)}
          data-value={value}
        />
      ),
      useRetrieve: vi.fn(() => ({
        data: {
          name: "mock-channel",
          dataType: { isNumeric: true, equals: () => false },
        },
      })),
    },
    Button: {
      ...((actual as Record<string, unknown>).Button as object),
      Button: ({
        children,
        onClick,
        disabled,
        tooltip,
      }: {
        children: React.ReactNode;
        onClick?: () => void;
        disabled?: boolean;
        size?: string;
        variant?: string;
        ghost?: boolean;
        tooltip?: string;
      }) => (
        <button
          data-testid={`btn-${tooltip?.toLowerCase().replace(/ /g, "-") ?? "unknown"}`}
          onClick={onClick}
          disabled={disabled}
        >
          {children}
        </button>
      ),
    },
    Flex: {
      Box: ({
        children,
        style,
      }: {
        children: React.ReactNode;
        x?: boolean;
        y?: boolean;
        style?: React.CSSProperties;
        gap?: string;
        className?: string;
        align?: string;
        full?: string;
      }) => (
        <div data-testid="flex-box" style={style}>
          {children}
        </div>
      ),
    },
    Icon: {
      ...((actual as Record<string, unknown>).Icon as object),
      Close: () => <span data-testid="icon-close" />,
    },
    Input: {
      Numeric: ({
        value,
        disabled,
      }: {
        value: number;
        onChange: (v: number) => void;
        disabled: boolean;
        resetValue?: number;
        bounds?: object;
        shrink?: boolean;
        variant?: string;
        tooltip?: string;
      }) => (
        <input
          data-testid="input-numeric"
          readOnly
          value={value}
          data-disabled={String(disabled)}
          onChange={() => {}}
        />
      ),
      Text: ({
        value,
        disabled,
        placeholder,
      }: {
        value: string;
        onChange: (v: string) => void;
        disabled?: boolean;
        placeholder?: string;
        variant?: string;
        shrink?: boolean;
      }) => (
        <input
          data-testid="input-text"
          readOnly
          value={value}
          data-disabled={String(disabled ?? false)}
          placeholder={placeholder}
          onChange={() => {}}
        />
      ),
    },
    Color: {
      Swatch: ({
        disabled,
      }: {
        value: unknown;
        onChange: (c: unknown) => void;
        onDelete?: () => void;
        size?: string;
        disabled: boolean;
      }) => <div data-testid="color-swatch" data-disabled={String(disabled)} />,
    },
  };
});

import * as Pluto from "@synnaxlabs/pluto";

import * as Selectors from "@/log/selectors";
import { Channels } from "@/log/toolbar/Channels";

const ZERO_STATE: State = {
  key: "test-key",
  version: "1.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  showChannelNames: true,
};

describe("log/toolbar/Channels", () => {
  it("renders null when state is null", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(undefined);
    const { container } = render(<Channels layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders the add-channel row when state is present and channels are empty", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByTestId("add-channel-select")).toBeDefined();
  });

  it("renders a row for each active channel", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      channels: [
        { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
        { channel: 20, color: "", notation: "standard", precision: -1, alias: "" },
      ],
    });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByTestId("channel-select-10")).toBeDefined();
    expect(screen.getByTestId("channel-select-20")).toBeDefined();
  });

  it("renders precision and color controls per channel row", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      channels: [
        { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
      ],
    });
    render(<Channels layoutKey="test-key" />);
    // One from the ChannelRow, one from the AddChannelRow
    expect(screen.getAllByTestId("input-numeric")).toHaveLength(2);
    expect(screen.getAllByTestId("color-swatch")).toHaveLength(2);
  });

  it("renders a remove button per channel row", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      channels: [
        { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
      ],
    });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByTestId("btn-remove-channel")).toBeDefined();
  });

  it("always renders the blank add-channel row at the bottom", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      channels: [
        { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
        { channel: 20, color: "", notation: "standard", precision: -1, alias: "" },
      ],
    });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByTestId("add-channel-select")).toBeDefined();
  });

  it("disables controls when user lacks edit permission", () => {
    vi.mocked(Pluto.Access.useUpdateGranted).mockReturnValueOnce(false);
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({
      ...ZERO_STATE,
      channels: [
        { channel: 10, color: "", notation: "standard", precision: -1, alias: "" },
      ],
    });
    render(<Channels layoutKey="test-key" />);
    expect(screen.getByTestId("channel-select-10").getAttribute("data-disabled")).toBe(
      "true",
    );
    expect(screen.getByTestId("add-channel-select").getAttribute("data-disabled")).toBe(
      "true",
    );
  });
});
