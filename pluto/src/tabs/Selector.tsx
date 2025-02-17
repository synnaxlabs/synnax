// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  type DragEventHandler,
  type MouseEventHandler,
  type ReactElement,
  useCallback,
} from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon as PIcon } from "@/icon";
import { Menu } from "@/menu";
import { useContext } from "@/tabs/Tabs";
import { type Spec } from "@/tabs/types";
import { Text } from "@/text";
import { type ComponentSize } from "@/util/component";

export interface SelectorProps
  extends Omit<Align.SpaceProps, "children" | "contextMenu"> {
  size?: ComponentSize;
  altColor?: boolean;
  contextMenu?: Menu.ContextMenuProps["menu"];
}

const CLS = "tabs-selector";

export const Selector = ({
  className,
  altColor = false,
  size = "medium",
  direction = "x",
  contextMenu,
  ...rest
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
  } = useContext();
  const menuProps = Menu.useContextMenu();
  const content = (
    <Align.Space
      className={CSS(CSS.B(CLS), CSS.size(size), className)}
      align="center"
      justify="spaceBetween"
      onDrop={onDrop}
      empty
      direction={direction}
      {...rest}
    >
      <Align.Space direction={direction} className={CSS.BE(CLS, "tabs")} empty>
        {tabs.map((tab) => (
          <SelectorButton
            key={tab.tabKey}
            selected={selected}
            altColor={altColor}
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
  if (contextMenu != null)
    return (
      <Menu.ContextMenu
        style={{ height: "fit-content" }}
        {...menuProps}
        menu={contextMenu}
      >
        {content}
      </Menu.ContextMenu>
    );
  return content;
};

const SelectorButton = ({
  selected,
  altColor = false,
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

  const handleClick = useCallback(() => onSelect?.(tabKey), [onSelect, tabKey]);

  const isSelected = selected === tabKey;
  const hasIcon = icon != null;

  const level = Text.ComponentSizeLevels[size];

  return (
    <Align.Pack
      size={size}
      id={tabKey}
      className={CSS(
        CSS.BE(CLS, "btn"),
        Menu.CONTEXT_TARGET,
        onRename == null && CSS.BEM(CLS, "btn", "uneditable"),
        isSelected && Menu.CONTEXT_SELECTED,
        CSS.selected(isSelected),
        CSS.altColor(altColor),
        closable && onClose != null && CSS.BEM(CLS, "btn", "closable"),
        hasIcon && CSS.BEM(CLS, "btn", "has-icon"),
        CSS.editable(onRename != null && editable),
      )}
      draggable
      direction="x"
      justify="center"
      align="center"
      onClick={handleClick}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      bordered={false}
      rounded={false}
    >
      {PIcon.resolve(icon as PIcon.Element, {
        className: CSS.BE(CLS, "icon"),
        style: {
          color: CSS.shadeVar(7),
          height: CSS.levelSizeVar(level),
          width: CSS.levelSizeVar(level),
        },
      })}
      <Name
        name={name}
        tabKey={tabKey}
        onRename={onRename}
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

export interface SelectorButtonProps extends Spec {
  selected?: string;
  altColor?: boolean;
  onDragStart?: (e: React.DragEvent<HTMLDivElement>, tab: Spec) => void;
  onDragEnd?: (e: React.DragEvent<HTMLDivElement>, tab: Spec) => void;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  onRename?: (key: string, name: string) => void;
  size: ComponentSize;
}

interface NameProps extends Text.CoreProps<Text.Level> {
  onRename?: (key: string, name: string) => void;
  name: string;
  tabKey: string;
  editable?: boolean;
}

const NAME_CLS = CSS.BE(CLS, "name");

const Name = ({
  onRename,
  name,
  tabKey,
  editable = true,
  ...rest
}: NameProps): ReactElement => {
  if (onRename == null || !editable)
    return (
      <Text.Text className={NAME_CLS} noWrap {...rest}>
        {name}
      </Text.Text>
    );
  return (
    <div className={NAME_CLS}>
      <Text.Editable<Text.Level>
        id={CSS.B(`tab-${tabKey}`)}
        onChange={(newText: string) => onRename(tabKey, newText)}
        value={name}
        noWrap
        {...rest}
      />
    </div>
  );
};
