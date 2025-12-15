// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type location, scale, xy } from "@synnaxlabs/x";
import {
  type DragEventHandler,
  type MouseEventHandler,
  type ReactElement,
  type ReactNode,
  useCallback,
  useState,
} from "react";

import { Button } from "@/button";
import { type Component } from "@/component";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { type Spec } from "@/tabs/types";
import { useContext } from "@/tabs/useContext";
import { Text } from "@/text";

export interface SelectorProps
  extends Omit<Flex.BoxProps, "children" | "contextMenu" | "onDrop"> {
  size?: Component.Size;
  altColor?: boolean;
  contextMenu?: Menu.ContextMenuProps["menu"];
  onDrop?: (e: React.DragEvent<HTMLElement>) => void;
  addTooltip?: string;
  actions?: ReactNode;
}

const CLS = "tabs-selector";

export const Selector = ({
  className,
  altColor = false,
  size = "medium",
  direction = "x",
  contextMenu,
  addTooltip,
  actions,
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
  const [draggingOver, setDraggingOver] = useState<boolean>(false);
  return (
    <>
      {contextMenu != null && (
        <Menu.ContextMenu
          style={{ height: "fit-content" }}
          {...menuProps}
          menu={contextMenu}
        />
      )}
      <Flex.Box
        className={CSS(
          CSS.B(CLS),
          className,
          menuProps.className,
          draggingOver && CSS.M("drag-over"),
        )}
        size={size}
        align="center"
        justify="between"
        empty
        direction={direction}
        onContextMenu={menuProps.open}
        onDrop={onDrop}
        {...rest}
      >
        <Flex.Box direction={direction} className={CSS.BE(CLS, "tabs")} empty>
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
          {onDrop != null && (
            <Flex.Box
              onDragOver={() => setDraggingOver(true)}
              onDragLeave={() => setDraggingOver(false)}
              onDragEnd={() => setDraggingOver(false)}
              onDrop={() => setDraggingOver(false)}
              grow
            />
          )}
        </Flex.Box>

        {(actions != null || onCreate != null) && (
          <Flex.Box className={CSS.BE(CLS, "actions")}>
            {onCreate != null && (
              <Button.Button
                size={size}
                sharp
                onClick={onCreate}
                tooltip={addTooltip}
                variant="text"
              >
                <Icon.Add />
              </Button.Button>
            )}
            {actions}
          </Flex.Box>
        )}
      </Flex.Box>
    </>
  );
};

interface CloseIconProps extends Icon.IconProps {
  unsavedChanges?: boolean;
}

const CloseIcon = ({ unsavedChanges, ...props }: CloseIconProps): ReactElement => {
  const closeIcon = <Icon.Close {...props} />;
  if (unsavedChanges)
    return (
      <>
        <Icon.Circle />
        {closeIcon}
      </>
    );
  return closeIcon;
};

const TABS_SELECTOR_BUTTON_CLASS = CSS.BE("tabs-selector", "btn");

const calculateDragOverPosition = (e: React.DragEvent<HTMLElement>): location.X => {
  if (!(e.target instanceof HTMLElement)) return "right";
  const closest = e.target.closest(`.${TABS_SELECTOR_BUTTON_CLASS}`);
  if (closest == null) return "right";
  const b = box.construct(closest);
  const cursor = xy.construct(e);
  const s = scale.Scale.scale(box.left(b), box.right(b)).scale(0, 1).pos(cursor.x);
  if (s < 0.5) return "left";
  return "right";
};

interface StartIconProps
  extends Icon.IconProps,
    Pick<SelectorButtonProps, "icon" | "loading"> {
  level: Text.Level;
}

const StartIcon = ({ loading, icon, level = "p" }: StartIconProps) => {
  if (loading) icon = <Icon.Loading />;
  return Icon.resolve(icon as Icon.ReactElement, {
    className: CSS.BE(CLS, "icon"),
    style: {
      color: CSS.colorVar(9),
      height: CSS.levelSizeVar(level),
      width: CSS.levelSizeVar(level),
    },
  });
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
  unsavedChanges = false,
  loading = false,
  onDrop,
}: SelectorButtonProps): ReactElement => {
  const handleDragStart: DragEventHandler<HTMLElement> = useCallback(
    (e) => onDragStart?.(e, { tabKey, name }),
    [onDragStart, tabKey, name],
  );

  const handleDragEnd: DragEventHandler<HTMLElement> = useCallback(
    (e) => onDragEnd?.(e, { tabKey, name }),
    [onDragEnd, tabKey, name],
  );

  const handleClose: MouseEventHandler<HTMLButtonElement> = useCallback(
    (e) => {
      e.preventDefault();
      e.stopPropagation();
      onClose?.(tabKey);
    },
    [onClose, tabKey],
  );

  const handleClick = useCallback(() => onSelect?.(tabKey), [onSelect, tabKey]);
  const [dragOverPosition, setDragOverPosition] = useState<location.X | null>(null);

  const handleDragOver: DragEventHandler<HTMLElement> = useCallback(
    (e) => {
      setDragOverPosition(calculateDragOverPosition(e));
    },
    [setDragOverPosition, onDrop],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLElement>) => {
      onDrop?.(e);
      setDragOverPosition(null);
    },
    [onDrop, setDragOverPosition],
  );

  const isSelected = selected === tabKey;
  const level = Text.COMPONENT_SIZE_LEVELS[size];

  return (
    <Button.Button
      el="div"
      size={size}
      id={tabKey}
      variant="text"
      sharp
      className={CSS(
        Menu.CONTEXT_TARGET,
        TABS_SELECTOR_BUTTON_CLASS,
        isSelected && Menu.CONTEXT_SELECTED,
        CSS.selected(isSelected),
        CSS.altColor(altColor),
        closable && onClose != null && CSS.M("closable"),
        CSS.editable(editable && onRename != null),
        dragOverPosition != null && CSS.M("drag-over"),
        dragOverPosition != null && CSS.loc(dragOverPosition),
      )}
      draggable
      justify="center"
      align="center"
      empty
      tabIndex={0}
      preventClick={isSelected}
      onClick={handleClick}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
      onDragLeave={() => setDragOverPosition(null)}
      onDragStart={handleDragStart}
      onDragEnd={(e) => {
        setDragOverPosition(null);
        handleDragEnd(e);
      }}
      bordered={false}
      rounded={false}
    >
      <StartIcon loading={loading} icon={icon} level={level} />
      <Name
        name={name}
        tabKey={tabKey}
        onRename={onRename}
        editable={editable}
        level={Text.COMPONENT_SIZE_LEVELS[size]}
      />
      {closable && onClose != null && (
        <Button.Button
          aria-label="pluto-tabs__close"
          onClick={handleClose}
          className={CSS.E("close")}
          variant="text"
          sharp
        >
          <CloseIcon unsavedChanges={unsavedChanges} />
        </Button.Button>
      )}
    </Button.Button>
  );
};

export interface SelectorButtonProps extends Spec {
  selected?: string;
  altColor?: boolean;
  onDragStart?: (e: React.DragEvent<HTMLElement>, tab: Spec) => void;
  onDragEnd?: (e: React.DragEvent<HTMLElement>, tab: Spec) => void;
  onDrop?: (e: React.DragEvent<HTMLElement>) => void;
  onSelect?: (key: string) => void;
  onClose?: (key: string) => void;
  onRename?: (key: string, name: string) => void;
  size: Component.Size;
}

interface NameProps extends Pick<Text.EditableProps, "level"> {
  onRename?: (key: string, name: string) => void;
  name: string;
  tabKey: string;
  editable?: boolean;
}

const Name = ({
  onRename,
  name,
  tabKey,
  editable = true,
  level,
}: NameProps): ReactElement => {
  if (onRename == null || !editable)
    return (
      <Text.Text overflow="ellipsis" level={level}>
        {name}
      </Text.Text>
    );
  return (
    <Text.Editable
      level={level}
      id={CSS.B(`tab-${tabKey}`)}
      onChange={(newText: string) => onRename?.(tabKey, newText)}
      value={name}
      overflow="ellipsis"
    />
  );
};
