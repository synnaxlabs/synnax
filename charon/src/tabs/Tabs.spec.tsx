// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Tabs } from "@/tabs";

const StaticTabs = ({ tabs, ...rest }: Tabs.TabsProps): ReactElement => {
  const props = Tabs.useStatic({ tabs });
  return <Tabs.Tabs {...rest} {...props} />;
};

describe("Tabs", () => {
  it("should render the first tabs content by default", () => {
    const tabs = [
      { tabKey: "tab1", name: "Tab 1", content: <span>Tab 1 Content</span> },
      { tabKey: "tab2", name: "Tab 2", content: <span>Tab 2 Content</span> },
    ];
    const { getByText } = render(<StaticTabs tabs={tabs} />);
    expect(getByText("Tab 1 Content")).toBeTruthy();
  });
  it("should switch the selected tab when clicked", () => {
    const tabs = [
      { tabKey: "tab1", name: "Tab 1", content: <span>Tab 1 Content</span> },
      { tabKey: "tab2", name: "Tab 2", content: <span>Tab 2 Content</span> },
    ];
    const { getByText } = render(<StaticTabs tabs={tabs} />);
    expect(getByText("Tab 1 Content")).toBeTruthy();
    expect(getByText("Tab 2")).toBeTruthy();
    fireEvent.click(getByText("Tab 2"));
    expect(getByText("Tab 2 Content")).toBeTruthy();
  });
  it("should render empty content when no tabs are provided", () => {
    const { getByText } = render(
      <StaticTabs tabs={[]} emptyContent={<h1>No Tabs</h1>} />,
    );
    expect(getByText("No Tabs")).toBeTruthy();
  });
  it("should render tabs using a render function if provided", () => {
    const { getByText } = render(
      <StaticTabs tabs={[{ tabKey: "tab1", name: "Tab 1" }]}>
        {(tab) => <h1>{tab.name} Content</h1>}
      </StaticTabs>,
    );
    expect(getByText("Tab 1 Content")).toBeTruthy();
  });
  it("should call the onTabDragStart and onTabDragEnd handler when a tab is dragged", () => {
    const handleDragStart = vi.fn();
    const handleDragEnd = vi.fn();
    const tabs: Tabs.Tab[] = [
      { tabKey: "tab1", name: "Tab 1", content: <span>Tab 1 Content</span> },
      { tabKey: "tab2", name: "Tab 2", content: <span>Tab 2 Content</span> },
    ];
    const { getByText } = render(
      <StaticTabs
        tabs={tabs}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      />,
    );
    expect(handleDragStart).not.toHaveBeenCalled();
    fireEvent.dragStart(getByText("Tab 1"));
    expect(handleDragStart).toHaveBeenCalled();
    fireEvent.dragEnd(getByText("Tab 1"));
    expect(handleDragEnd).toHaveBeenCalled();
  });
  it("should render a close button if an onClose prop is passed", () => {
    const onClose = vi.fn();
    const tabs = [
      { tabKey: "tab1", name: "Tab 1", content: <span>Tab 1 Content</span> },
    ];
    const { getByText, getByLabelText } = render(
      <StaticTabs tabs={tabs} onClose={onClose} />,
    );
    expect(getByText("Tab 1")).toBeTruthy();
    const btn = getByLabelText("pluto-tabs__close");
    expect(btn).toBeTruthy();
    fireEvent.click(getByLabelText("pluto-tabs__close"));
    expect(onClose).toHaveBeenCalled();
  });
});
