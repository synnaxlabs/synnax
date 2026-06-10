// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ComponentType, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Tabs } from "@/tabs";
import { type NameProps } from "@/tabs/types";

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
  it("should make tabs draggable only when an onDragStart handler is provided", () => {
    const tabs: Tabs.Tab[] = [{ tabKey: "tab1", name: "Tab 1" }];
    const getTabButton = (container: HTMLElement): HTMLElement => {
      const btn = container.querySelector<HTMLElement>(".pluto-tabs-selector__btn");
      if (btn == null) throw new Error("tab button not found");
      return btn;
    };
    const withHandler = render(<StaticTabs tabs={tabs} onDragStart={vi.fn()} />);
    expect(getTabButton(withHandler.container).draggable).toBe(true);
    const withoutHandler = render(<StaticTabs tabs={tabs} />);
    expect(getTabButton(withoutHandler.container).draggable).toBe(false);
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

  describe("default tab name rendering", () => {
    it("should commit a rename via the default name when Enter is pressed", () => {
      const onRename = vi.fn();
      const tabs: Tabs.Tab[] = [{ tabKey: "tab1", name: "Tab 1" }];
      const { getByText } = render(<StaticTabs tabs={tabs} onRename={onRename} />);
      const tab = getByText("Tab 1");
      fireEvent.dblClick(tab);
      tab.innerText = "Renamed";
      fireEvent.keyDown(tab, { key: "Enter" });
      expect(onRename).toHaveBeenCalledWith("tab1", "Renamed");
    });

    it("should not allow editing when no onRename prop is provided", () => {
      const tabs: Tabs.Tab[] = [{ tabKey: "tab1", name: "Tab 1" }];
      const { getByText } = render(<StaticTabs tabs={tabs} />);
      const tab = getByText("Tab 1");
      fireEvent.dblClick(tab);
      expect(tab.getAttribute("contenteditable")).not.toBe("true");
    });

    it("should not allow editing a tab whose editable flag is false", () => {
      const onRename = vi.fn();
      const tabs: Tabs.Tab[] = [{ tabKey: "tab1", name: "Tab 1", editable: false }];
      const { getByText } = render(<StaticTabs tabs={tabs} onRename={onRename} />);
      const tab = getByText("Tab 1");
      fireEvent.dblClick(tab);
      expect(tab.getAttribute("contenteditable")).not.toBe("true");
    });
  });

  describe("custom Name component", () => {
    it("should render a custom Name component when one is provided", () => {
      const CustomName: ComponentType<NameProps> = ({ name, tabKey, editable }) => (
        <span>{`name=${name} key=${tabKey} editable=${editable ?? true}`}</span>
      );
      const tabs: Tabs.Tab[] = [{ tabKey: "tab1", name: "Tab 1", editable: false }];
      const { getByText } = render(<StaticTabs tabs={tabs} Name={CustomName} />);
      expect(getByText("name=Tab 1 key=tab1 editable=false")).toBeTruthy();
    });

    it("should pass the Tabs onRename through to the custom Name component", () => {
      const onRename = vi.fn();
      const CustomName: ComponentType<NameProps> = ({ tabKey, onRename }) => (
        <button onClick={() => onRename?.(tabKey, "Renamed")}>rename-trigger</button>
      );
      const tabs: Tabs.Tab[] = [{ tabKey: "tab1", name: "Tab 1" }];
      const { getByText } = render(
        <StaticTabs tabs={tabs} Name={CustomName} onRename={onRename} />,
      );
      fireEvent.click(getByText("rename-trigger"));
      expect(onRename).toHaveBeenCalledWith("tab1", "Renamed");
    });
  });
});
