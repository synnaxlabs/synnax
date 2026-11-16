// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/tree/item.css";

import { type resource, type Synnax } from "@synnaxlabs/client";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { type FC, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/platform/css";
import { type Children } from "@/platform/tree/children";
import { MatchGroup, useHighlight } from "@/platform/tree/search";
import { type ContextMenu, type Resource } from "@/platform/tree/types";

// ItemProps are the props the Tree passes to a resource type's Item. The Item owns its
// own icon and double-click behavior, so neither is threaded through here.
export interface ItemProps extends Omit<Base.ItemProps<string>, "id" | "resource"> {
  resource: Resource;
  loading: boolean;
}

// ContentProps extend ItemProps with the icon and double-click handler the factory
// resolves. A custom Content receives these; the default row renders them directly.
export interface ContentProps extends ItemProps {
  icon?: Icon.ReactElement;
  onDoubleClick: () => void;
}

export interface Content extends FC<ContentProps> {}

export interface CanDropProps extends Haul.DraggingState {
  /** The root of the tree. */
  root: resource.ID;
  /** The resource the items would drop onto. */
  destination: Resource;
  /** The tree ancestors of the destination, nearest first, ending with the root. */
  ancestors: resource.ID[];
  /** The tree items the drop would move. */
  moved: Dropped[];
}

/** Decides whether a node accepts the dragged items. */
export interface CanDrop {
  (props: CanDropProps): boolean;
}

export interface Dropped {
  id: resource.ID;
  /** The tree ancestors of the item, nearest first, ending with the tree root. */
  ancestors: resource.ID[];
}

export interface DropProps {
  client: Synnax;
  destination: Resource;
  items: Dropped[];
}

/** Moves the dropped items into a node. */
export interface Drop {
  (props: DropProps): Promise<void>;
}

/** Returns what a drag of the resource carries besides the resource itself. */
export interface HaulItems {
  (resource: Resource, client: Synnax): Haul.Item[];
}

/**
 * Item renders one resource in the resource tree. The component carries the static
 * behavior of its resource type (children, order, drag, context menu) as properties,
 * which the Tree reads imperatively.
 */
export interface Item extends FC<ItemProps> {
  hasChildren: boolean;
  /** Whether nodes of this type show before the other children of their parent. */
  folder: boolean;
  children?: Children;
  canDrop: CanDrop;
  drop?: Drop;
  haulItems: HaulItems;
  ContextMenu?: ContextMenu;
  /** A deep link opens a resource of this type. */
  linkable: boolean;
  /** Returns the icon that the item shows for the resource. */
  icon: (resource: Resource) => Icon.ReactElement | undefined;
  /** Renders its children with the handler that a double click on the item runs. */
  OnSelect: FC<OnSelectProps>;
}

export interface OnSelectProps {
  children: (onSelect: (resource: Resource) => void) => ReactNode;
}

export interface Items extends Partial<Record<resource.Type, Item>> {}

const DefaultRow = ({
  onDoubleClick,
  resource,
  icon,
  loading,
  ...rest
}: ContentProps): ReactElement => {
  const highlight = useHighlight();
  return (
    <Base.Item {...rest} onDoubleClick={onDoubleClick}>
      {icon}
      <Text.MaybeEditable
        id={List.itemNameID(resource.key)}
        value={resource.name}
        onChange
        allowDoubleClick={false}
        className={CSS.BE("tree-item", "name")}
        overflow="fade"
        highlight={highlight}
      />
      <MatchGroup id={resource.id} />
    </Base.Item>
  );
};

const noop = (): void => {};

export interface CreateItemParams {
  icon?: Icon.ReactElement | ((resource: Resource) => Icon.ReactElement);
  useOnSelect?: () => (resource: Resource) => void;
  /** Lists the children of a node of this type. */
  children?: Children;
  /** Whether a node of this type expands. Defaults to whether it lists children. */
  hasChildren?: boolean;
  /** Whether nodes of this type show before the other children of their parent. */
  folder?: boolean;
  canDrop?: CanDrop;
  /** Moves dropped items into a node of this type. Required when canDrop can pass. */
  drop?: Drop;
  haulItems?: HaulItems;
  ContextMenu?: ContextMenu;
  /** A deep link opens a resource of this type. */
  linkable?: boolean;
  Content?: Content;
}

export const createItem = ({
  icon,
  useOnSelect = () => noop,
  children,
  hasChildren = children != null,
  folder = false,
  canDrop = () => false,
  drop,
  haulItems = () => [],
  ContextMenu,
  linkable = false,
  Content: ContentComp,
}: CreateItemParams): Item => {
  const Row = ContentComp ?? DefaultRow;
  const resolveIcon = (resource: Resource): Icon.ReactElement | undefined =>
    Icon.resolve(typeof icon === "function" ? icon(resource) : icon);
  const Component = (props: ItemProps): ReactElement => {
    const onSelect = useOnSelect();
    const handleDoubleClick = () => onSelect(props.resource);
    return (
      <Row
        {...props}
        icon={resolveIcon(props.resource)}
        onDoubleClick={handleDoubleClick}
      />
    );
  };
  const OnSelect = ({ children }: OnSelectProps): ReactNode => {
    const onSelect = useOnSelect();
    return children(onSelect);
  };
  const item = Component as Item;
  item.icon = resolveIcon;
  item.OnSelect = OnSelect;
  item.hasChildren = hasChildren;
  item.folder = folder;
  item.children = children;
  item.canDrop = canDrop;
  item.drop = drop;
  item.haulItems = haulItems;
  item.ContextMenu = ContextMenu;
  item.linkable = linkable;
  return item;
};

// DefaultItem is the fallback used for resource types with no registered Item: an
// unadorned name row with no children and no drag or drop behavior.
export const DefaultItem: Item = createItem({});
