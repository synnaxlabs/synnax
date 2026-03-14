// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { fireEvent, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Log } from "@/log";
import { type State } from "@/log/types/v0";
import { renderWithConsole } from "@/testUtils";

vi.mock("@/log/export", () => ({
  useExport: vi.fn(() => vi.fn()),
  extract: vi.fn(),
}));

vi.mock("@/export", () => ({
  Export: {
    ToolbarButton: ({ onExport }: { onExport: () => void }) => (
      <button data-testid="export-button" onClick={onExport}>
        Export
      </button>
    ),
  },
}));

vi.mock("@/cluster", () => ({
  Cluster: {
    CopyLinkToolbarButton: ({ name }: { name: string; ontologyID?: unknown }) => (
      <button data-testid="copy-link-button" data-name={name}>
        Copy Link
      </button>
    ),
  },
}));

vi.mock("@/log/toolbar/Channels", () => ({
  Channels: ({ layoutKey }: { layoutKey: string }) => (
    <div data-testid="channels-tab" data-layout-key={layoutKey} />
  ),
}));

vi.mock("@/log/toolbar/Properties", () => ({
  Properties: ({ layoutKey }: { layoutKey: string }) => (
    <div data-testid="properties-tab" data-layout-key={layoutKey} />
  ),
}));

vi.mock("@/log/Toolbar.css", () => ({}));

vi.mock("@/css", () => ({
  CSS: {
    B: (name: string) => name,
  },
}));

vi.mock("@/components", () => ({
  Toolbar: {
    Content: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="toolbar-content">{children}</div>
    ),
    Header: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="toolbar-header">{children}</div>
    ),
    Title: ({
      children,
      icon,
    }: {
      children: React.ReactNode;
      icon?: React.ReactNode;
    }) => (
      <div data-testid="toolbar-title">
        {icon}
        {children}
      </div>
    ),
  },
}));

vi.mock("@synnaxlabs/pluto", async (importOriginal) => {
  const actual = await importOriginal();

  const TabsContext = React.createContext<{
    tabs: Array<{ tabKey: string; name: string }>;
    selected: string;
    content: (tab: { tabKey: string }) => React.ReactNode;
    onSelect: (key: string) => void;
  } | null>(null);
  TabsContext.displayName = "TabsContext";

  const Provider = ({
    value,
    children,
  }: {
    value: {
      tabs: Array<{ tabKey: string; name: string }>;
      selected: string;
      content: (tab: { tabKey: string }) => React.ReactNode;
      onSelect: (key: string) => void;
    };
    children: React.ReactNode;
  }) => <TabsContext value={value}>{children}</TabsContext>;

  const Selector = ({ style }: { style?: React.CSSProperties }) => {
    const ctx = React.useContext(TabsContext);
    if (ctx == null) return null;
    return (
      <div data-testid="tabs-selector" style={style}>
        {ctx.tabs.map((t) => (
          <button
            key={t.tabKey}
            data-testid={`tab-${t.tabKey}`}
            data-selected={String(ctx.selected === t.tabKey)}
            onClick={() => ctx.onSelect(t.tabKey)}
          >
            {t.name}
          </button>
        ))}
      </div>
    );
  };

  const Content = () => {
    const ctx = React.useContext(TabsContext);
    if (ctx == null) return null;
    return (
      <div data-testid="tabs-content">{ctx.content({ tabKey: ctx.selected })}</div>
    );
  };

  return {
    ...(actual as object),
    Flex: {
      Box: ({ children }: { children: React.ReactNode }) => (
        <div data-testid="flex-box">{children}</div>
      ),
    },
    Icon: {
      ...((actual as Record<string, unknown>).Icon as object),
      Log: () => <span data-testid="icon-log" />,
    },
    Tabs: {
      ...((actual as Record<string, unknown>).Tabs as object),
      Provider,
      Selector,
      Content,
    },
  };
});

import * as LogExport from "@/log/export";
import { Toolbar } from "@/log/Toolbar";

const LAYOUT_KEY = "test-key";

const ZERO_LOG_STATE: State = {
  key: LAYOUT_KEY,
  version: "0.0.0",
  channels: [],
  remoteCreated: false,
  timestampPrecision: 0,
  channelConfigs: {},
  showChannelNames: true,
};

const LAYOUT_STATE: Layout.State = {
  key: LAYOUT_KEY,
  windowKey: MAIN_WINDOW,
  type: "log",
  name: "Test Log",
  location: "mosaic",
};

const preloadState = (
  logState: State = ZERO_LOG_STATE,
  layoutState: Layout.State = LAYOUT_STATE,
) => ({
  [Layout.SLICE_NAME]: {
    ...Layout.ZERO_SLICE_STATE,
    layouts: { ...Layout.ZERO_SLICE_STATE.layouts, [logState.key]: layoutState },
  },
  [Log.SLICE_NAME]: { ...Log.ZERO_SLICE_STATE, logs: { [logState.key]: logState } },
});

const renderToolbar = (layoutKey: string = LAYOUT_KEY, logState?: State) => {
  const ls = logState ?? ZERO_LOG_STATE;
  const lay: Layout.State = { ...LAYOUT_STATE, key: layoutKey };
  return renderWithConsole(<Toolbar layoutKey={layoutKey} />, {
    preloadedState: preloadState({ ...ls, key: layoutKey }, lay),
  });
};

describe("log/Toolbar", () => {
  it("renders null when log state does not exist in store", () => {
    // Layout entry exists but log state doesn't — Toolbar should render null.
    const layoutOnly: Layout.State = {
      ...LAYOUT_STATE,
      key: "no-log",
    };
    const { container } = renderWithConsole(<Toolbar layoutKey="no-log" />, {
      preloadedState: {
        [Layout.SLICE_NAME]: {
          ...Layout.ZERO_SLICE_STATE,
          layouts: { ...Layout.ZERO_SLICE_STATE.layouts, "no-log": layoutOnly },
        },
        [Log.SLICE_NAME]: Log.ZERO_SLICE_STATE,
      },
    });
    expect(container.firstChild).toBeNull();
  });

  it("renders the toolbar content when state is present", () => {
    renderToolbar();
    expect(screen.getByTestId("toolbar-content")).toBeDefined();
  });

  it("displays the layout name in the title", () => {
    renderWithConsole(<Toolbar layoutKey={LAYOUT_KEY} />, {
      preloadedState: preloadState(ZERO_LOG_STATE, { ...LAYOUT_STATE, name: "My Log" }),
    });
    expect(screen.getByText("My Log")).toBeDefined();
  });

  it("renders both tab buttons (Channels and Properties)", () => {
    renderToolbar();
    expect(screen.getByTestId("tab-channels")).toBeDefined();
    expect(screen.getByTestId("tab-properties")).toBeDefined();
  });

  it("defaults to the channels tab", () => {
    renderToolbar();
    expect(screen.getByTestId("tab-channels").getAttribute("data-selected")).toBe(
      "true",
    );
    expect(screen.getByTestId("tab-properties").getAttribute("data-selected")).toBe(
      "false",
    );
    expect(screen.getByTestId("channels-tab")).toBeDefined();
  });

  it("switches to the Properties tab when the properties tab button is clicked", () => {
    renderToolbar();
    fireEvent.click(screen.getByTestId("tab-properties"));
    expect(screen.getByTestId("tab-properties").getAttribute("data-selected")).toBe(
      "true",
    );
    expect(screen.getByTestId("properties-tab")).toBeDefined();
  });

  it("switches back to channels tab", () => {
    renderToolbar();
    fireEvent.click(screen.getByTestId("tab-properties"));
    fireEvent.click(screen.getByTestId("tab-channels"));
    expect(screen.getByTestId("tab-channels").getAttribute("data-selected")).toBe(
      "true",
    );
    expect(screen.getByTestId("channels-tab")).toBeDefined();
  });

  it("renders export and copy-link toolbar buttons", () => {
    renderToolbar();
    expect(screen.getByTestId("export-button")).toBeDefined();
    expect(screen.getByTestId("copy-link-button")).toBeDefined();
  });

  it("copy-link button receives the layout name", () => {
    renderWithConsole(<Toolbar layoutKey={LAYOUT_KEY} />, {
      preloadedState: preloadState(ZERO_LOG_STATE, {
        ...LAYOUT_STATE,
        name: "CopyName",
      }),
    });
    expect(screen.getByTestId("copy-link-button").getAttribute("data-name")).toBe(
      "CopyName",
    );
  });

  it("calls handleExport when export button is clicked", () => {
    const mockHandleExport = vi.fn();
    vi.mocked(LogExport.useExport).mockReturnValue(mockHandleExport);
    renderToolbar();
    fireEvent.click(screen.getByTestId("export-button"));
    expect(mockHandleExport).toHaveBeenCalledWith(LAYOUT_KEY);
  });

  it("passes the layoutKey to sub-components", () => {
    renderToolbar("my-layout", { ...ZERO_LOG_STATE, key: "my-layout" });
    expect(screen.getByTestId("channels-tab").getAttribute("data-layout-key")).toBe(
      "my-layout",
    );
  });
});
