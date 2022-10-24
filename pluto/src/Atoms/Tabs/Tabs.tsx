import { ComponentType, ReactElement, useEffect, useState } from "react";
import clsx from "clsx";
import { Space, SpaceProps } from "../Space";
import { Text } from "../Typography";
import { Button } from "../Button";
import { AiOutlineClose } from "react-icons/ai";
import "./Tabs.css";

export interface TabsProps
  extends Omit<SpaceProps, "children" | "onSelect" | "size"> {
  tabs: TabEntry[];
  selected?: string;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  closable?: boolean;
  emptyContent?: ReactElement | null;
  children?: ComponentType<{ tab: TabEntry }> | null;
  onTabDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabEntry) => void;
}

export interface TabEntry {
  tabKey: string;
  title: string;
  content?: ReactElement;
  closable?: boolean;
}

const Tabs = ({
  onSelect: propsOnSelect,
  onClose: propsOnClose,
  selected: propsSelected,
  onTabDragStart,
  closable = true,
  tabs: propsTabs = [],
  emptyContent = null,
  className,
  children: Children,
  onDrag,
  ...props
}: TabsProps) => {
  const [tabs, setTabs] = useState<TabEntry[]>(propsTabs);
  const [selected, setSelected] = useState<string>(
    propsSelected || tabs[0]?.tabKey || ""
  );

  useEffect(() => {
    setTabs(propsTabs);
  }, [propsTabs]);

  useEffect(() => {
    if (!propsTabs.map((t) => t.tabKey).includes(selected)) {
      setSelected(propsTabs[0]?.tabKey || "");
    }
  }, [tabs]);

  useEffect(() => {
    if (propsSelected) setSelected(propsSelected);
  }, [propsSelected]);

  const selectedTab = tabs.find((tab) => tab.tabKey === selected);

  const onSelect = (key: string) => {
    setSelected(key);
    propsOnSelect?.(key);
  };

  const onClose = (key: string) => {
    setTabs((tabs) => tabs.filter((tab) => tab.tabKey !== key));
    propsOnClose?.(key);
  };

  let content: ReactElement;
  if (selectedTab && Children) {
    content = <Children tab={selectedTab} />;
  } else {
    content = selectedTab?.content ?? emptyContent ?? <></>;
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
          <Tab
            key={tab.tabKey}
            selected={selected}
            onSelect={onSelect}
            closable={closable}
            onClose={onClose}
            onTabDragStart={onTabDragStart}
            {...tab}
          />
        ))}
      </Space>
      {content}
    </Space>
  );
};

export interface TabProps extends TabEntry {
  selected?: string;
  onTabDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabEntry) => void;
  onSelect: (key: string) => void;
  onClose: (key: string) => void;
}

const Tab = ({
  selected,
  onSelect,
  onClose,
  tabKey,
  title,
  closable,
  onTabDragStart,
}: TabProps) => {
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
      onClick={() => onSelect(tabKey)}
      onDragStart={(e) =>
        onTabDragStart && onTabDragStart(e, { tabKey, title })
      }
    >
      <Text level="p">{title}</Text>
      {closable && (
        <Button.IconOnly
          size="small"
          onClick={(e) => {
            e.stopPropagation();
            onClose(tabKey);
          }}
          style={{ height: "3rem", padding: "1rem 0.25rem" }}
        >
          <AiOutlineClose />
        </Button.IconOnly>
      )}
    </Space>
  );
};

export default Tabs;
