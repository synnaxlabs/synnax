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

// ---------------------------------------------------------------------------
// Module mocks
// ---------------------------------------------------------------------------

vi.mock("@/log/selectors", () => ({
  useSelectOptional: vi.fn(),
}));

vi.mock("@/layout", () => ({
  Layout: {
    useSelectRequired: vi.fn(() => ({ name: "Test Log" })),
  },
}));

vi.mock("@/log/export", () => ({
  useExport: vi.fn(() => vi.fn()),
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

// Stub sub-toolbar components so Toolbar tests stay isolated
vi.mock("@/log/toolbar/Channels", () => ({
  Channels: ({ layoutKey }: { layoutKey: string }) => (
    <div data-testid="channels-tab" data-layout-key={layoutKey} />
  ),
}));

vi.mock("@/log/toolbar/Text", () => ({
  Text: ({ layoutKey }: { layoutKey: string }) => (
    <div data-testid="text-tab" data-layout-key={layoutKey} />
  ),
}));

// Stub CSS to avoid missing CSS file errors
vi.mock("@/log/Toolbar.css", () => ({}));

vi.mock("@/css", () => ({
  CSS: {
    B: (name: string) => name,
  },
}));

// Minimal Toolbar / Tabs stubs — enough to render content and simulate tab switching
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

  // A tiny controllable Tabs implementation
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
      Provider,
      Selector,
      Content,
    },
  };
});

// ---------------------------------------------------------------------------
// Import component under test
// ---------------------------------------------------------------------------

import { Layout } from "@/layout";
import * as LogExport from "@/log/export";
import * as Selectors from "@/log/selectors";
import { Toolbar } from "@/log/Toolbar";

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

describe("log/Toolbar", () => {
  it("renders null when state is null", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(undefined);
    const { container } = render(<Toolbar layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders null when state is null-ish", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue(
      null as unknown as undefined,
    );
    const { container } = render(<Toolbar layoutKey="test-key" />);
    expect(container.firstChild).toBeNull();
  });

  it("renders the toolbar content when state is present", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    expect(screen.getByTestId("toolbar-content")).toBeDefined();
  });

  it("displays the layout name in the title", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    vi.mocked(Layout.useSelectRequired).mockReturnValue({
      name: "My Log",
    } as ReturnType<typeof Layout.useSelectRequired>);
    render(<Toolbar layoutKey="test-key" />);
    expect(screen.getByText("My Log")).toBeDefined();
  });

  it("renders both tab buttons (Channels and Text)", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    expect(screen.getByTestId("tab-channels")).toBeDefined();
    expect(screen.getByTestId("tab-text")).toBeDefined();
  });

  it("defaults to the channels tab", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    expect(screen.getByTestId("tab-channels").getAttribute("data-selected")).toBe(
      "true",
    );
    expect(screen.getByTestId("tab-text").getAttribute("data-selected")).toBe("false");
    expect(screen.getByTestId("channels-tab")).toBeDefined();
  });

  it("switches to the Text tab when the text tab button is clicked", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    fireEvent.click(screen.getByTestId("tab-text"));
    expect(screen.getByTestId("tab-text").getAttribute("data-selected")).toBe("true");
    expect(screen.getByTestId("text-tab")).toBeDefined();
  });

  it("switches back to channels tab", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    fireEvent.click(screen.getByTestId("tab-text"));
    fireEvent.click(screen.getByTestId("tab-channels"));
    expect(screen.getByTestId("tab-channels").getAttribute("data-selected")).toBe(
      "true",
    );
    expect(screen.getByTestId("channels-tab")).toBeDefined();
  });

  it("renders export and copy-link toolbar buttons", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    expect(screen.getByTestId("export-button")).toBeDefined();
    expect(screen.getByTestId("copy-link-button")).toBeDefined();
  });

  it("copy-link button receives the layout name", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    vi.mocked(Layout.useSelectRequired).mockReturnValue({
      name: "CopyName",
    } as ReturnType<typeof Layout.useSelectRequired>);
    render(<Toolbar layoutKey="test-key" />);
    expect(screen.getByTestId("copy-link-button").getAttribute("data-name")).toBe(
      "CopyName",
    );
  });

  it("calls handleExport when export button is clicked", () => {
    const mockHandleExport = vi.fn();
    vi.mocked(LogExport.useExport).mockReturnValue(mockHandleExport);
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="test-key" />);
    fireEvent.click(screen.getByTestId("export-button"));
    expect(mockHandleExport).toHaveBeenCalledWith("test-key");
  });

  it("passes the layoutKey to sub-components", () => {
    vi.mocked(Selectors.useSelectOptional).mockReturnValue({ ...ZERO_STATE });
    render(<Toolbar layoutKey="my-layout" />);
    expect(screen.getByTestId("channels-tab").getAttribute("data-layout-key")).toBe(
      "my-layout",
    );
  });
});
