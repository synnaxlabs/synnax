// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { type State } from "@/log/types/v1";
import { renderWithConsole } from "@/testUtils";

const mockDispatch = vi.fn();

vi.mock("@/log/Log", async (importOriginal) => {
  const actual = await importOriginal();
  return { ...(actual as object), useSyncComponent: () => mockDispatch };
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

import { Properties } from "@/log/toolbar/Properties";

const LOG_STATE: State = {
  key: "test-key",
  version: "1.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  showChannelNames: true,
};

const preloadState = (logState: State) => ({
  [Layout.SLICE_NAME]: {
    ...Layout.ZERO_SLICE_STATE,
    layouts: {
      [logState.key]: {
        key: logState.key,
        name: "Test Log",
        type: "log",
        location: "mosaic" as const,
        icon: "Log",
        window: undefined,
        tab: undefined,
        windowKey: logState.key,
      },
    },
  },
  [Log.SLICE_NAME]: {
    ...Log.ZERO_SLICE_STATE,
    logs: { [logState.key]: logState },
  },
});

describe("log/toolbar/Properties", () => {
  it("renders null when state is null", () => {
    const { container } = renderWithConsole(<Properties layoutKey="nonexistent" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    expect(container.firstChild).toBeNull();
  });

  it("renders timestamp precision and show channel names controls", () => {
    renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    expect(screen.getByText("Receipt Timestamp Precision")).toBeDefined();
    expect(screen.getByText("Show Channel Names")).toBeDefined();
  });

  it("renders the current timestamp precision value", () => {
    renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState({ ...LOG_STATE, timestampPrecision: 2 }),
    });
    const input = screen.getByTestId("input-numeric");
    expect(Number((input as HTMLInputElement).value)).toBe(2);
  });

  it("renders the current showChannelNames value", () => {
    renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState({ ...LOG_STATE, showChannelNames: false }),
    });
    expect(screen.getByTestId("input-switch").getAttribute("data-checked")).toBe(
      "false",
    );
  });

  it("dispatches setShowChannelNames when toggle is clicked", () => {
    renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    fireEvent.click(screen.getByTestId("input-switch"));
    expect(mockDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "log/setShowChannelNames",
        payload: { key: "test-key", showChannelNames: false },
      }),
    );
  });

  it("disables controls when user lacks edit permission", () => {
    vi.mocked(Pluto.Access.useUpdateGranted).mockReturnValueOnce(false);
    renderWithConsole(<Properties layoutKey="test-key" />, {
      preloadedState: preloadState(LOG_STATE),
    });
    expect(screen.getByTestId("input-numeric").getAttribute("data-disabled")).toBe(
      "true",
    );
    expect(screen.getByTestId("input-switch").getAttribute("data-disabled")).toBe(
      "true",
    );
  });
});
