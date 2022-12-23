import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Tabs, TabsProps } from ".";

const StaticTabs = ({ tabs, ...tProps }: TabsProps): JSX.Element => {
  const props = Tabs.useStatic({ tabs });
  return <Tabs {...tProps} {...props} />;
};

describe("Tabs", () => {
  it("should render the first tabs content by default", () => {
    const tabs = [
      { tabKey: "tab1", title: "Tab 1", content: "Tab 1 Content" },
      { tabKey: "tab2", title: "Tab 2", content: "Tab 2 Content" },
    ];
    const { getByText } = render(<StaticTabs tabs={tabs} />);
    expect(getByText("Tab 1 Content")).toBeTruthy();
  });
  it("should switch the selected tab when clicked", () => {
    const tabs = [
      { tabKey: "tab1", title: "Tab 1", content: "Tab 1 Content" },
      { tabKey: "tab2", title: "Tab 2", content: "Tab 2 Content" },
    ];
    const { getByText } = render(<StaticTabs tabs={tabs} />);
    expect(getByText("Tab 1 Content")).toBeTruthy();
    expect(getByText("Tab 2")).toBeTruthy();
    fireEvent.click(getByText("Tab 2"));
    expect(getByText("Tab 2 Content")).toBeTruthy();
  });
  it("should render empty content when no tabs are provided", () => {
    const { getByText } = render(
      <StaticTabs tabs={[]} emptyContent={<h1>No Tabs</h1>} />
    );
    expect(getByText("No Tabs")).toBeTruthy();
  });
  it("should render tabs using a render function if provided", () => {
    const { getByText } = render(
      <StaticTabs tabs={[{ tabKey: "tab1", title: "Tab 1" }]}>
        {({ tab }) => <h1>{tab.title} Content</h1>}
      </StaticTabs>
    );
    expect(getByText("Tab 1 Content")).toBeTruthy();
  });
  it("should call the onTabDragStart and onTabDragEnd handler when a tab is dragged", () => {
    const onTabDragStart = vi.fn();
    const onTabDragEnd = vi.fn();
    const tabs = [
      { tabKey: "tab1", title: "Tab 1", content: "Tab 1 Content" },
      { tabKey: "tab2", title: "Tab 2", content: "Tab 2 Content" },
    ];
    const { getByText } = render(
      <StaticTabs
        tabs={tabs}
        onTabDragStart={onTabDragStart}
        onTabDragEnd={onTabDragEnd}
      />
    );
    expect(onTabDragStart).not.toHaveBeenCalled();
    fireEvent.dragStart(getByText("Tab 1"));
    expect(onTabDragStart).toHaveBeenCalled();
    fireEvent.dragEnd(getByText("Tab 1"));
    expect(onTabDragEnd).toHaveBeenCalled();
  });
  it("should render a close button if an onClose prop is passed", () => {
    const onClose = vi.fn();
    const tabs = [{ tabKey: "tab1", title: "Tab 1", content: "Tab 1 Content" }];
    const { getByText, getByLabelText } = render(
      <StaticTabs tabs={tabs} onClose={onClose} />
    );
    expect(getByText("Tab 1")).toBeTruthy();
    const btn = getByLabelText("pluto-tabs__close");
    expect(btn).toBeTruthy();
    fireEvent.click(getByLabelText("pluto-tabs__close"));
    expect(onClose).toHaveBeenCalled();
  });
});
