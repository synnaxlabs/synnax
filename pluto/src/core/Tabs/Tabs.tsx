// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, { createContext, ReactElement, useContext, useState } from "react";

import { Space, SpaceProps } from "@/core/Space";
import { TabMeta, TabsSelector } from "@/core/Tabs/TabsSelector";
import { CSS } from "@/css";
import { ComponentSize } from "@/util/component";
import { RenderProp } from "@/util/renderProp";

export interface Tab extends TabMeta {
  content?: JSX.Element;
}

export type TabRenderProp = RenderProp<Tab>;

export interface UseStaticTabsProps {
  tabs: Tab[];
  content?: TabRenderProp;
}

export const resetTabSelection = (
  selected = "",
  tabs: Tab[] = []
): string | undefined => {
  if (tabs.length === 0) return undefined;
  return tabs.find((t) => t.tabKey === selected) != null
    ? selected
    : tabs[tabs.length - 1]?.tabKey;
};

export const renameTab = (key: string, title: string, tabs: Tab[]): Tab[] => {
  title = title.trim();
  if (title.length === 0) return tabs;
  const t = tabs.find((t) => t.tabKey === key);
  if (t == null || t.name === title) return tabs;
  return tabs.map((t) => (t.tabKey === key ? { ...t, name: title } : t));
};

export const useStaticTabs = ({
  tabs,
  content,
}: UseStaticTabsProps): TabsContextValue => {
  const [selected, setSelected] = useState(tabs[0]?.tabKey ?? "");

  return {
    tabs,
    selected,
    content,
    onSelect: setSelected,
  };
};

export interface TabsContextValue {
  tabs: Tab[];
  emptyContent?: ReactElement | null;
  closable?: boolean;
  selected?: string;
  onSelect?: (key: string) => void;
  content?: TabRenderProp;
  onClose?: (key: string) => void;
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabMeta) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: TabMeta) => void;
  onDrop?: (e: React.DragEvent<HTMLDivElement>) => void;
  onRename?: (key: string, title: string) => void;
  onCreate?: () => void;
}

export interface TabsProps
  extends Omit<
      SpaceProps,
      "children" | "onSelect" | "size" | "onDragStart" | "onDragEnd"
    >,
    TabsContextValue {
  children?: TabRenderProp;
  size?: ComponentSize;
}

export const TabsContext = createContext<TabsContextValue>({ tabs: [] });

export const useTabsContext = (): TabsContextValue => useContext(TabsContext);

export const Tabs = ({
  content,
  children,
  onSelect,
  selected,
  closable,
  tabs,
  onClose,
  onDragStart,
  onDragEnd,
  onCreate,
  onRename,
  emptyContent,
  className,
  onDragOver,
  onDrop,
  size = "medium",
  ...props
}: TabsProps): JSX.Element => (
  <Space
    empty
    className={CSS(CSS.B("tabs"), className)}
    onDragOver={onDragOver}
    onDrop={onDrop}
    {...props}
  >
    <TabsContext.Provider
      value={{
        tabs,
        emptyContent,
        selected,
        closable,
        content: children ?? content,
        onSelect,
        onClose,
        onDragStart,
        onDragEnd,
        onRename,
        onCreate,
        onDrop,
      }}
    >
      <TabsSelector size={size} />
      <TabsContent />
    </TabsContext.Provider>
  </Space>
);

export const TabsContent = (): JSX.Element | null => {
  const { tabs, selected, content: renderProp, emptyContent } = useTabsContext();
  let content: JSX.Element | string | null = null;
  const selectedTab = tabs.find((tab) => tab.tabKey === selected);
  if (selectedTab != null) {
    if (renderProp != null) content = renderProp(selectedTab);
    else if (selectedTab.content != null) content = selectedTab.content;
  } else if (emptyContent != null) content = emptyContent;
  return content;
};
