// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, type IconProps } from "@synnaxlabs/media";
import { box, type location, scale, xy } from "@synnaxlabs/x";
import {
  type DragEventHandler,
  type MouseEventHandler,
  type ReactElement,
  useCallback,
  useState,
} from "react";

import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon as PIcon } from "@/icon";
import { Menu } from "@/menu";
import { type Spec } from "@/tabs/types";
import { useContext } from "@/tabs/useContext";
import { Text } from "@/text";
import { type ComponentSize } from "@/util/component";

export interface SelectorProps
  extends Omit<Align.SpaceProps, "children" | "contextMenu" | "onDrop"> {
  size?: ComponentSize;
  altColor?: boolean;
  contextMenu?: Menu.ContextMenuProps["menu"];
  onDrop?: (e: React.DragEvent<HTMLDivElement>) => void;
  addTooltip?: string;
}

const CLS = "tabs-selector";

export const Selector = ({
  className,
  altColor = false,
  size = "medium",
  direction = "x",
  contextMenu,
  addTooltip,
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
      <Align.Space
        className={CSS(
          CSS.B(CLS),
          CSS.size(size),
          className,
          menuProps.className,
          draggingOver && CSS.M("drag-over"),
        )}
        align="center"
        justify="spaceBetween"
        empty
        direction={direction}
        onContextMenu={menuProps.open}
        onDrop={onDrop}
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
          {onDrop != null && (
            <Align.Space
              onDragOver={() => setDraggingOver(true)}
              onDragLeave={() => setDraggingOver(false)}
              onDragEnd={() => setDraggingOver(false)}
              onDrop={() => setDraggingOver(false)}
              grow
            />
          )}
        </Align.Space>

        {onCreate != null && (
          <Align.Space className={CSS.BE(CLS, "actions")}>
            <Button.Icon size={size} sharp onClick={onCreate} tooltip={addTooltip}>
              <Icon.Add />
            </Button.Icon>
          </Align.Space>
        )}
      </Align.Space>
    </>
  );
};

interface CloseIconProps extends IconProps {
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

const calculateDragOverPosition = (e: React.DragEvent<HTMLDivElement>): location.X => {
  const b = box.construct(
    (e.target as HTMLElement).closest(".pluto-tabs-selector__btn"),
  );
  const cursor = xy.construct(e);
  const s = scale.Scale.scale(box.left(b), box.right(b)).scale(0, 1).pos(cursor.x);
  if (s < 0.5) return "left";
  return "right";
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
  onDrop,
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
      e.preventDefault();
      e.stopPropagation();
      onClose?.(tabKey);
    },
    [onClose, tabKey],
  );

  const handleClick = useCallback(() => onSelect?.(tabKey), [onSelect, tabKey]);
  const [dragOverPosition, setDragOverPosition] = useState<location.X | null>(null);

  const handleDragOver: DragEventHandler<HTMLDivElement> = useCallback(
    (e) => {
      setDragOverPosition(calculateDragOverPosition(e));
    },
    [setDragOverPosition, onDrop],
  );

  const handleDrop = useCallback(
    (e: React.DragEvent<HTMLDivElement>) => {
      onDrop?.(e);
      setDragOverPosition(null);
    },
    [onDrop, setDragOverPosition],
  );

  const isSelected = selected === tabKey;
  const level = Text.ComponentSizeLevels[size];

  return (
    <Align.Pack
      size={size}
      id={tabKey}
      className={CSS(
        CSS.BE(CLS, "btn"),
        Menu.CONTEXT_TARGET,
        isSelected && Menu.CONTEXT_SELECTED,
        CSS.selected(isSelected),
        CSS.altColor(altColor),
        closable && onClose != null && CSS.M("closable"),
        CSS.editable(editable && onRename != null),
        dragOverPosition != null && CSS.M("drag-over"),
        dragOverPosition != null && CSS.loc(dragOverPosition),
      )}
      draggable
      x
      justify="center"
      align="center"
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
      {PIcon.resolve(icon as PIcon.Element, {
        className: CSS.BE(CLS, "icon"),
        style: {
          color: CSS.shadeVar(9),
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
        <Button.Icon
          aria-label="pluto-tabs__close"
          onClick={handleClose}
          className={CSS.E("close")}
        >
          <CloseIcon unsavedChanges={unsavedChanges} />
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
  onDrop?: (e: React.DragEvent<HTMLDivElement>) => void;
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

const Name = ({
  onRename,
  name,
  tabKey,
  editable = true,
  ...rest
}: NameProps): ReactElement => {
  if (onRename == null || !editable)
    return (
      <Text.Text noWrap {...rest}>
        {name}
      </Text.Text>
    );
  return (
    <Text.Editable<Text.Level>
      id={CSS.B(`tab-${tabKey}`)}
      onChange={(newText: string) => onRename(tabKey, newText)}
      value={name}
      noWrap
      {...rest}
    />
  );
};
