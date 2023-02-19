// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import clsx from "clsx";
import { AiOutlineClose } from "react-icons/ai";

import { Button } from "@/core/Button";
import { Space, SpaceProps } from "@/core/Space";

import { useTabsContext } from "./Tabs";

import { Text } from "@/core/Typography";

import "./TabsSelector.css";

export interface TabMeta {
  tabKey: string;
  name: string;
  closable?: boolean;
  icon?: JSX.Element;
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
    onRename,
  } = useTabsContext();
  return tabs.length > 0 ? (
    <Space
      className={clsx("pluto-tabs-selector", className)}
      direction="x"
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
          onRename={onRename}
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
  name,
  onTabDragStart,
  onTabDragEnd,
  onRename,
  closable,
  icon,
}: TabSelectorButtonProps): JSX.Element => {
  const onDragStart = (e: React.DragEvent<HTMLDivElement>): void =>
    onTabDragStart?.(e, { tabKey, name });

  const onDragEnd = (e: React.DragEvent<HTMLDivElement>): void =>
    onTabDragEnd?.(e, { tabKey, name });

  const _onClose = (e: React.MouseEvent<HTMLButtonElement>): void => {
    e.stopPropagation();
    onClose?.(tabKey);
  };

  const _onSelect = (): void => onSelect?.(tabKey);

  return (
    <Space
      className={clsx(
        "pluto-tabs-selector__button",
        onRename == null && "pluto-tabs-selector__button--uneditable",
        selected === tabKey && "pluto-tabs-selector__button--selected",
        closable === true && "pluto-tabs-selector__button--closable"
      )}
      draggable
      direction="x"
      justify="center"
      align="center"
      onClick={_onSelect}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
    >
      <TabName name={name} tabKey={tabKey} onRename={onRename} icon={icon} />
      {onClose != null && (
        <Button.Icon
          size="small"
          onClick={_onClose}
          style={{ height: "3rem", padding: "1rem 0.25rem" }}
        >
          <AiOutlineClose aria-label="pluto-tabs__close" />
        </Button.Icon>
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
  onRename?: (key: string, name: string) => void;
}

interface TabnameProps {
  onRename?: (key: string, name: string) => void;
  name: string;
  tabKey: string;
  icon?: JSX.Element;
}

const TabName = ({ onRename, name, tabKey, icon }: TabnameProps): JSX.Element => {
  if (onRename == null) {
    if (icon != null)
      return (
        <Text.WithIcon startIcon={icon} level="p">
          {name}
        </Text.WithIcon>
      );
    return <Text level="p">{name}</Text>;
  }
  return (
    <Text.Editable level="p" onChange={(newText: string) => onRename(tabKey, newText)}>
      {name}
    </Text.Editable>
  );
};
