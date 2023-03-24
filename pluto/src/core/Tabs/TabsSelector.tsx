// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { useTabsContext } from "./Tabs";

import { Button } from "@/core/Button";
import { Space, SpaceProps } from "@/core/Space";
import { Text } from "@/core/Typography";
import { CSS } from "@/css";
import { ComponentSize } from "@/util/component";

import "./TabsSelector.css";

export interface TabMeta {
  tabKey: string;
  name: string;
  closable?: boolean;
  icon?: JSX.Element;
  editable?: boolean;
}

export interface TabsSelectorProps extends Omit<SpaceProps, "children" | "size"> {
  size: ComponentSize;
}

const CLS = "tabs-selector";

export const TabsSelector = ({
  className,
  size = "medium",
  ...props
}: TabsSelectorProps): JSX.Element | null => {
  const {
    tabs,
    selected,
    onSelect,
    onClose,
    closable,
    onDragEnd,
    onDragStart,
    onDragOver,
    onDrop,
    onRename,
    onCreate,
  } = useTabsContext();
  return (
    <Space
      className={CSS(CSS.B(CLS), CSS.size(size), className)}
      direction="x"
      align="center"
      justify="start"
      onDragOver={onDragOver}
      onDrop={onDrop}
      empty
      {...props}
    >
      {tabs.map((tab) => (
        <TabC
          key={tab.tabKey}
          selected={selected}
          onSelect={onSelect}
          onClose={onClose}
          onDragStart={onDragStart}
          onDragEnd={onDragEnd}
          onRename={onRename}
          closable={tab.closable ?? closable}
          size={size}
          {...tab}
        />
      ))}
      <Space className={CSS.BE(CLS, "actions")}>
        {onCreate != null && (
          <Button.Icon size={size} sharp onClick={onCreate}>
            <Icon.Add />
          </Button.Icon>
        )}
      </Space>
    </Space>
  );
};

const TabC = ({
  selected,
  onSelect,
  onClose,
  tabKey,
  name,
  onDragStart,
  onDragEnd,
  onRename,
  closable,
  icon,
  size,
  editable = true,
}: TabSelectorButtonProps): JSX.Element => {
  const ohandleDragStart = (e: React.DragEvent<HTMLDivElement>): void =>
    onDragStart?.(e, { tabKey, name });

  const handleDragEnd = (e: React.DragEvent<HTMLDivElement>): void =>
    onDragEnd?.(e, { tabKey, name });

  const handleClose = (e: React.MouseEvent<HTMLButtonElement>): void => {
    e.stopPropagation();
    onClose?.(tabKey);
  };

  const _onSelect = (): void => onSelect?.(tabKey);

  return (
    <Space
      className={CSS(
        CSS.size(size),
        CSS.BE(CLS, "btn"),
        onRename == null && CSS.BEM(CLS, "button", "uneditable"),
        CSS.selected(selected === tabKey),
        closable === true && onClose != null && CSS.BEM(CLS, "button", "closable")
      )}
      draggable
      direction="x"
      justify="center"
      align="center"
      onClick={_onSelect}
      onDragStart={ohandleDragStart}
      onDragEnd={handleDragEnd}
    >
      <TabName
        name={name}
        tabKey={tabKey}
        onRename={onRename}
        icon={icon}
        editable={editable}
      />
      {onClose != null && (
        <Button.Icon
          size="small"
          onClick={handleClose}
          style={{ height: "3.5rem", padding: "1rem 0.75rem" }}
        >
          <Icon.Close aria-label="pluto-tabs__close" />
        </Button.Icon>
      )}
    </Space>
  );
};

export interface TabSelectorButtonProps extends TabMeta {
  selected?: string;
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabMeta) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: TabMeta) => void;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  onRename?: (key: string, name: string) => void;
  size: ComponentSize;
}

interface TabnameProps {
  onRename?: (key: string, name: string) => void;
  name: string;
  tabKey: string;
  icon?: JSX.Element;
  editable?: boolean;
}

const TabName = ({
  onRename,
  name,
  tabKey,
  icon,
  editable = true,
}: TabnameProps): JSX.Element => {
  if (onRename == null || !editable) {
    if (icon != null)
      return (
        <Text.WithIcon startIcon={icon} level="p" noWrap>
          {name}
        </Text.WithIcon>
      );
    return <Text level="p">{name}</Text>;
  }
  return (
    <Text.Editable<"p">
      level="p"
      onChange={(newText: string) => onRename(tabKey, newText)}
      value={name}
      noWrap
    />
  );
};
