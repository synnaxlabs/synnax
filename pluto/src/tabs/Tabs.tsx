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
  useCallback,
} from "react";

import { direction } from "@synnaxlabs/x";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { state } from "@/state";
import { type TabSpec, Selector } from "@/tabs/Selector";
import { type ComponentSize } from "@/util/component";
import { type RenderProp } from "@/util/renderProp";

import "@/tabs/Tabs.css";

export interface Tab extends TabSpec {
  content?: ReactNode;
}

export type TabRenderProp = RenderProp<Tab>;

export interface UseStaticTabsProps {
  tabs: Tab[];
  content?: TabRenderProp;
  onSelect?: (key: string) => void;
  selected?: string;
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
  selected,
  onSelect,
}: UseStaticTabsProps): TabsContextValue => {
  const [value, onChange] = state.usePurePassthrough({
    initial: selected ?? tabs[0]?.tabKey ?? "",
    value: selected,
    onChange: onSelect,
  });
  const valueRef = useSyncedRef(selected ?? value);

  const handleSelect = useCallback(
    (key: string): void => {
      onChange(key);
      if (valueRef.current == null) onSelect?.(key);
    },
    [value, onSelect],
  );

  return {
    tabs,
    selected: value,
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
  content?: TabRenderProp | ReactNode;
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
  children?: TabRenderProp | ReactNode;
  size?: ComponentSize;
}

export const TabsContext = createContext<TabsContextValue>({ tabs: [] });

export const useTabsContext = (): TabsContextValue => useContext(TabsContext);

export const Tabs = ({
  id,
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
  direction: dir = "y",
  ...props
}: TabsProps): ReactElement => (
  <Align.Space
    id={id}
    empty
    className={CSS(CSS.B("tabs"), className)}
    onDragOver={onDragOver}
    onDrop={onDrop}
    direction={dir}
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
      <Selector size={size} direction={direction.swap(dir)} />
      <Content />
    </TabsContext.Provider>
  </Align.Space>
);

export const Provider = TabsContext.Provider;

export const Content = (): ReactNode | null => {
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
  if (renderProp != null) {
    if (typeof renderProp === "function") content = renderProp(selectedTab);
    else content = renderProp;
  } else if (selectedTab.content != null) content = selectedTab.content;
  return (
    <div className={CSS.B("tabs-content")} onClick={() => onSelect?.(selected)}>
      {content}
    </div>
  );
};
