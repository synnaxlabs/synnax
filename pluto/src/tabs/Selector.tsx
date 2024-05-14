// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type DragEventHandler,
  type MouseEventHandler,
  type ReactElement,
  useCallback,
} from "react";

import { Icon } from "@synnaxlabs/media";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useTabsContext } from "@/tabs/Tabs";
import { Text } from "@/text";
import { type ComponentSize } from "@/util/component";

import "@/tabs/Selector.css";

export interface TabSpec {
  tabKey: string;
  name: string;
  closable?: boolean;
  icon?: ReactElement;
  editable?: boolean;
}

export interface SelectorProps extends Omit<Align.SpaceProps, "children"> {
  size?: ComponentSize;
}

const CLS = "tabs-selector";

export const Selector = ({
  className,
  size = "medium",
  direction = "x",
  ...props
}: SelectorProps): ReactElement | null => {
  const {
    tabs,
    selected,
    onSelect,
    onClose,
    closable,
    onDragEnd,
    onDragStart,
    onDrop,
    onRename,
    onCreate,
  } = useTabsContext();
  return (
    <Align.Space
      className={CSS(CSS.B(CLS), CSS.size(size), className)}
      align="center"
      justify="spaceBetween"
      onDrop={onDrop}
      empty
      direction={direction}
      {...props}
    >
      <Align.Space direction={direction} className={CSS.BE(CLS, "tabs")} empty>
        {tabs.map((tab) => (
          <SelectorButton
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
      </Align.Space>

      {onCreate != null && (
        <Align.Space className={CSS.BE(CLS, "actions")}>
          <Button.Icon size={size} sharp onClick={onCreate}>
            <Icon.Add />
          </Button.Icon>
        </Align.Space>
      )}
    </Align.Space>
  );
};

const SelectorButton = ({
  selected,
  onSelect,
  onClose,
  tabKey,
  name,
  onDragStart,
  onDragEnd,
  onRename,
  closable = true,
  icon,
  size,
  editable = true,
}: SelectorButtonProps): ReactElement => {
  const handleDragStart: DragEventHandler<HTMLDivElement> = useCallback(
    (e) => onDragStart?.(e, { tabKey, name }),
    [onDragStart, tabKey, name],
  );

  const handleDragEnd: DragEventHandler<HTMLDivElement> = useCallback(
    (e) => onDragEnd?.(e, { tabKey, name }),
    [onDragEnd, tabKey, name],
  );

  const handleClose: MouseEventHandler<HTMLButtonElement> = useCallback(
    (e) => {
      e.stopPropagation();
      onClose?.(tabKey);
    },
    [onClose, tabKey],
  );

  const _onSelect = useCallback(() => onSelect?.(tabKey), [onSelect, tabKey]);

  return (
    <Align.Pack
      size={size}
      className={CSS(
        CSS.BE(CLS, "btn"),
        onRename == null && CSS.BEM(CLS, "btn", "uneditable"),
        CSS.selected(selected === tabKey),
        closable && onClose != null && CSS.BEM(CLS, "btn", "closable"),
      )}
      draggable
      direction="x"
      justify="center"
      align="center"
      onClick={_onSelect}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      bordered={false}
      rounded={false}
    >
      <Name
        name={name}
        tabKey={tabKey}
        onRename={onRename}
        icon={icon}
        editable={editable}
        level={Text.ComponentSizeLevels[size]}
      />
      {closable && onClose != null && (
        <Button.Icon onClick={handleClose}>
          <Icon.Close aria-label="pluto-tabs__close" />
        </Button.Icon>
      )}
    </Align.Pack>
  );
};

export interface SelectorButtonProps extends TabSpec {
  selected?: string;
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: TabSpec) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: TabSpec) => void;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  onRename?: (key: string, name: string) => void;
  size: ComponentSize;
}

interface NameProps extends Text.CoreProps<Text.Level> {
  onRename?: (key: string, name: string) => void;
  name: string;
  tabKey: string;
  icon?: ReactElement;
  editable?: boolean;
}

const Name = ({
  onRename,
  name,
  tabKey,
  icon,
  editable = true,
  ...props
}: NameProps): ReactElement => {
  if (onRename == null || !editable) {
    if (icon != null)
      return (
        <Text.WithIcon startIcon={icon} noWrap {...props}>
          {name}
        </Text.WithIcon>
      );
    return (
      <Text.Text noWrap {...props}>
        {name}
      </Text.Text>
    );
  }
  return (
    <Text.Editable<Text.Level>
      onChange={(newText: string) => onRename(tabKey, newText)}
      value={name}
      noWrap
      {...props}
    />
  );
};
