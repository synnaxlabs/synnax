import React, { ComponentType, ReactNode, useState } from "react";

import clsx from "clsx";
import { AiOutlineClose } from "react-icons/ai";

import { Space, SpaceProps } from "../Space";

import { Button } from "@/atoms/Button";
import { Text } from "@/atoms/Typography";
import "./Tabs.css";

export interface TabsProps extends Omit<SpaceProps, "children" | "onSelect"> {
  tabs: Tab[];
  selected?: string;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  emptyContent?: ReactNode | ComponentType | null;
  children?: ComponentType<{ tab: Tab }> | null;
  onTabDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: Tab) => void;
  onTabDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: Tab) => void;
  onTitleChange?: (key: string, title: string) => void;
}

export interface Tab {
  tabKey: string;
  title: string;
  content?: ReactNode;
  closable?: boolean;
}

export interface UseStaticTabsProps {
  tabs: Tab[];
}

export const resetTabSelection = (
  selected = "",
  tabs: Tab[] = []
): string | undefined =>
  tabs.find((t) => t.tabKey === selected) != null ? selected : tabs[0]?.tabKey;

export const renameTab = (key: string, title: string, tabs: Tab[]): Tab[] => {
  title = title.trim();
  if (title.length === 0) return tabs;
  const t = tabs.find((t) => t.tabKey === key);
  if (t == null || t.title === title) return tabs;
  return tabs.map((t) => (t.tabKey === key ? { ...t, title } : t));
};

export const useStaticTabs = ({ tabs }: UseStaticTabsProps): TabsProps => {
  const [selected, setSelected] = useState(tabs[0]?.tabKey ?? "");

  return {
    tabs,
    selected,
    onSelect: setSelected,
  };
};

export const Tabs = ({
  onSelect,
  onClose,
  selected,
  onTabDragStart,
  onTabDragEnd,
  tabs = [],
  emptyContent: EmptyContent = null,
  className,
  children: Children,
  onTitleChange,
  ...props
}: TabsProps): JSX.Element => {
  let content = null;
  const selectedTab = tabs.find((tab) => tab.tabKey === selected);
  if (selectedTab != null) {
    if (Children != null) content = <Children tab={selectedTab} />;
    else if (selectedTab.content != null) content = selectedTab.content;
  } else if (tabs.length === 0 && EmptyContent != null)
    content = typeof EmptyContent === "function" ? <EmptyContent /> : EmptyContent;

  return (
    <Space className={clsx("pluto-tabs__container", className)} empty {...props}>
      {tabs.length > 0 && (
        <Space
          className="pluto-tabs__tabs"
          direction="horizontal"
          align="center"
          justify="start"
          empty
        >
          {tabs.map((tab) => (
            <TabC
              key={tab.tabKey}
              selected={selected}
              onSelect={onSelect}
              onClose={onClose}
              onTabDragStart={onTabDragStart}
              onTabDragEnd={onTabDragEnd}
              onTitleChange={onTitleChange}
              {...tab}
            />
          ))}
        </Space>
      )}
      {content}
    </Space>
  );
};

export interface TabProps extends Tab {
  selected?: string;
  onTabDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: Tab) => void;
  onTabDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: Tab) => void;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  onTitleChange?: (key: string, title: string) => void;
}

const TabC = ({
  selected,
  onSelect,
  onClose,
  tabKey,
  title,
  onTabDragStart,
  onTabDragEnd,
  onTitleChange,
}: TabProps): JSX.Element => {
  const onDragStart = (e: React.DragEvent<HTMLDivElement>): void =>
    onTabDragStart?.(e, { tabKey, title });

  const onDragEnd = (e: React.DragEvent<HTMLDivElement>): void =>
    onTabDragEnd?.(e, { tabKey, title });

  const _onClose = (e: React.MouseEvent<HTMLButtonElement>): void => {
    e.stopPropagation();
    onClose?.(tabKey);
  };

  const _onSelect = (): void => onSelect?.(tabKey);

  return (
    <Space
      className={clsx(
        "pluto-tabs__tab",
        onTitleChange == null && "pluto-tabs__tab--uneditable",
        selected === tabKey && "pluto-tabs__tab--selected"
      )}
      draggable
      direction="horizontal"
      justify="center"
      align="center"
      onClick={_onSelect}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
    >
      <TabTitle title={title} tabKey={tabKey} onTitleChange={onTitleChange} />
      {onClose != null && (
        <Button.IconOnly
          size="small"
          onClick={_onClose}
          style={{ height: "3rem", padding: "1rem 0.25rem" }}
        >
          <AiOutlineClose aria-label="pluto-tabs__close" />
        </Button.IconOnly>
      )}
    </Space>
  );
};

interface TabTitleProps {
  onTitleChange?: (key: string, title: string) => void;
  title: string;
  tabKey: string;
}

const TabTitle = ({ onTitleChange, title, tabKey }: TabTitleProps): JSX.Element => {
  if (onTitleChange == null) return <Text level="p">{title}</Text>;
  return (
    <Text.Editable
      level="p"
      onChange={(newText: string) => onTitleChange(tabKey, newText)}
    >
      {title}
    </Text.Editable>
  );
};
