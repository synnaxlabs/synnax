// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import React, {
  createContext,
  type ReactElement,
  type ReactNode,
  useContext,
  useState,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { type TabSpec, Selector } from "@/tabs/Selector";
import { type ComponentSize } from "@/util/component";
import { type RenderProp } from "@/util/renderProp";

export interface Tab extends TabSpec {
  content?: ReactNode;
}

export type TabRenderProp = RenderProp<Tab>;

export interface UseStaticTabsProps {
  tabs: Tab[];
  content?: TabRenderProp;
  onSelect?: (key: string) => void;
}

export const resetSelection = (selected = "", tabs: Tab[] = []): string | undefined => {
  if (tabs.length === 0) return undefined;
  return tabs.find((t) => t.tabKey === selected) != null
    ? selected
    : tabs[tabs.length - 1]?.tabKey;
};

export const rename = (key: string, title: string, tabs: Tab[]): Tab[] => {
  title = title.trim();
  if (title.length === 0) return tabs;
  const t = tabs.find((t) => t.tabKey === key);
  if (t == null || t.name === title) return tabs;
  return tabs.map((t) => (t.tabKey === key ? { ...t, name: title } : t));
};

export const useStatic = ({
  tabs,
  content,
  onSelect,
}: UseStaticTabsProps): TabsContextValue => {
  const [selected, setSelected] = useState(tabs[0]?.tabKey ?? "");

  const handleSelect = (key: string): void => {
    setSelected(key);
    onSelect?.(key);
  };

  return {
    tabs,
    selected,
    content,
    onSelect: handleSelect,
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
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabSpec) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: TabSpec) => void;
  onDrop?: (e: React.DragEvent<HTMLDivElement>) => void;
  onRename?: (key: string, title: string) => void;
  onCreate?: () => void;
}

export interface TabsProps
  extends Omit<
      Align.SpaceProps,
      "children" | "onSelect" | "size" | "onDragStart" | "onDragEnd" | "content"
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
}: TabsProps): ReactElement => (
  <Align.Space
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
      <Selector size={size} />
      <Content />
    </TabsContext.Provider>
  </Align.Space>
);

export const Provider = TabsContext.Provider;

export const Content = (): ReactElement | null => {
  const {
    tabs,
    selected,
    content: renderProp,
    emptyContent,
    onSelect,
  } = useTabsContext();
  let content: ReactNode = null;
  const selectedTab = tabs.find((tab) => tab.tabKey === selected);
  if (selected == null || selectedTab == null) return emptyContent ?? null;
  if (renderProp != null) content = renderProp(selectedTab);
  else if (selectedTab.content != null) content = selectedTab.content;
  return (
    <div onClick={() => onSelect?.(selected)} style={{ width: "100%", height: "100%" }}>
      {content}
    </div>
  );
};
