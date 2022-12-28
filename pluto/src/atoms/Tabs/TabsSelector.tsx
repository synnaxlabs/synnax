import clsx from "clsx";
import { AiOutlineClose } from "react-icons/ai";

import { Button } from "../Button";
import { Space, SpaceProps } from "../Space";
import { Text } from "../Typography";

import { useTabsContext } from "./Tabs";

import "./TabsSelector.css";

export interface TabMeta {
  tabKey: string;
  title: string;
  closable?: boolean;
}

export interface TabsSelectorProps extends Omit<SpaceProps, "children"> {}

export const TabsSelector = ({
  className,
  ...props
}: TabsSelectorProps): JSX.Element | null => {
  const {
    tabs,
    selected,
    onSelect,
    onClose,
    closable,
    onTabDragEnd,
    onTabDragStart,
    onTitleChange,
  } = useTabsContext();
  return tabs.length > 0 ? (
    <Space
      className={clsx("pluto-tabs-selector", className)}
      direction="horizontal"
      align="center"
      justify="start"
      empty
      {...props}
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
          closable={tab.closable ?? closable}
          {...tab}
        />
      ))}
    </Space>
  ) : null;
};

const TabC = ({
  selected,
  onSelect,
  onClose,
  tabKey,
  title,
  onTabDragStart,
  onTabDragEnd,
  onTitleChange,
  closable,
}: TabSelectorButtonProps): JSX.Element => {
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
        "pluto-tabs-selector__button",
        onTitleChange == null && "pluto-tabs-selector__button--uneditable",
        selected === tabKey && "pluto-tabs-selector__button--selected",
        closable === true && "pluto-tabs-selector__button--closable"
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

export interface TabSelectorButtonProps extends TabMeta {
  selected?: string;
  onTabDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabMeta) => void;
  onTabDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: TabMeta) => void;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  onTitleChange?: (key: string, title: string) => void;
}

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
