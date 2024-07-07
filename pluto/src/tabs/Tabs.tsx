// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tabs/Tabs.css";

import { direction } from "@synnaxlabs/x";
import React, {
  createContext,
  type ReactElement,
  type ReactNode,
  useCallback,
  useContext,
} from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { state } from "@/state";
import { Selector } from "@/tabs/Selector";
import { RenderProp, type Spec, Tab } from "@/tabs/types";
import { type ComponentSize } from "@/util/component";

export interface UseStaticTabsProps {
  tabs: Tab[];
  content?: RenderProp;
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
  content?: RenderProp | ReactNode;
  onClose?: (key: string) => void;
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: Spec) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: Spec) => void;
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
  children?: RenderProp | ReactNode;
  size?: ComponentSize;
  selectedAltColor?: boolean;
}

export const TabsContext = createContext<TabsContextValue>({ tabs: [] });

export const useTabsContext = (): TabsContextValue => useContext(TabsContext);

export const Tabs = ({
  id,
  content,
  children,
  onSelect,
  selected,
  selectedAltColor,
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
      <Selector
        size={size}
        direction={direction.swap(dir)}
        altColor={selectedAltColor}
      />
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
  } else if (selectedTab.content != null) content = selectedTab.content as ReactNode;
  return (
    <div className={CSS.B("tabs-content")} onClick={() => onSelect?.(selected)}>
      {content}
    </div>
  );
};
