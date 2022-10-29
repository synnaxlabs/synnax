import React, { ComponentType, ReactNode, useState } from "react";
import clsx from "clsx";
import { Space, SpaceProps } from "../Space";
import { Text } from "@/atoms/Typography";
import { Button } from "@/atoms/Button";
import { AiOutlineClose } from "react-icons/ai";
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

export const resetTabSelection = (selected: string = "", tabs: Tab[] = []) =>
  tabs.find((t) => t.tabKey === selected) ? selected : tabs[0]?.tabKey;

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
  onDrag,
  onTitleChange,
  ...props
}: TabsProps) => {
  let content = null;
  const selectedTab = tabs.find((tab) => tab.tabKey === selected);
  if (selectedTab) {
    if (Children) {
      content = <Children tab={selectedTab} />;
    } else if (selectedTab.content) {
      content = selectedTab.content;
    }
  } else if (tabs.length === 0 && EmptyContent) {
    content =
      typeof EmptyContent === "function" ? <EmptyContent /> : EmptyContent;
  }

  return (
    <Space
      className={clsx("pluto-tabs__container", className)}
      empty
      {...props}
    >
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
}: TabProps) => {
  const onDragStart = (e: React.DragEvent<HTMLDivElement>) => {
    if (onTabDragStart) onTabDragStart(e, { tabKey, title });
  };

  const onDragEnd = (e: React.DragEvent<HTMLDivElement>) => {
    if (onTabDragEnd) onTabDragEnd(e, { tabKey, title });
  };

  const _onClose = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    if (onClose) onClose(tabKey);
  };

  const _onSelect = () => onSelect && onSelect(tabKey);

  return (
    <Space
      className={clsx(
        "pluto-tabs__tab",
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
      <Text.Editable
        level="p"
        text={title}
        onChange={(newText) => {
          if (onTitleChange) onTitleChange(tabKey, newText);
        }}
      />
      {onClose && (
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
