// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/tree/item.css";

import { type ontology, type resource, type Synnax } from "@synnaxlabs/client";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Text } from "@synnaxlabs/lyra/text";
import { Tree as Base } from "@synnaxlabs/lyra/tree";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { type Children } from "@/platform/tree/children";
import { type ContextMenu } from "@/platform/tree/types";

// ItemProps are the props the Tree passes to a resource type's Item. The Item owns its
// own icon and double-click behavior, so neither is threaded through here.
export interface ItemProps extends Omit<Base.ItemProps<string>, "id" | "resource"> {
  resource: ontology.Resource;
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
  destination: ontology.Resource;
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
  destination: ontology.Resource;
  items: Dropped[];
}

/** Moves the dropped items into a node. */
export interface Drop {
  (props: DropProps): Promise<void>;
}

// Item renders a single resource row in the ontology tree. The component carries the
// static, render-independent behavior for its resource type (children, drag, context
// menu, visibility) as properties, read imperatively by the Tree.
export interface Item extends FC<ItemProps> {
  type: resource.Type;
  hasChildren: boolean;
  children?: Children;
  canDrop: CanDrop;
  drop?: Drop;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
  ContextMenu?: ContextMenu;
  visible?: (resource: ontology.Resource) => boolean;
}

export interface Items extends Partial<Record<resource.Type, Item>> {}

const DefaultRow = ({
  onDoubleClick,
  resource,
  icon,
  loading,
  ...rest
}: ContentProps): ReactElement => (
  <Base.Item {...rest} onDoubleClick={onDoubleClick}>
    {icon}
    <Text.MaybeEditable
      id={List.itemNameID(resource.key)}
      value={resource.name}
      onChange
      allowDoubleClick={false}
      className={CSS.BE("tree-item", "name")}
      overflow="fade"
    />
  </Base.Item>
);

const noop = (): void => {};

export interface CreateItemParams {
  type: resource.Type;
  icon?: Icon.ReactElement | ((resource: ontology.Resource) => Icon.ReactElement);
  useOnSelect?: () => (resource: ontology.Resource) => void;
  /** Lists the children of a node of this type. */
  children?: Children;
  /** Whether a node of this type expands. Defaults to whether it lists children. */
  hasChildren?: boolean;
  canDrop?: CanDrop;
  /** Moves dropped items into a node of this type. Required when canDrop can pass. */
  drop?: Drop;
  haulItems?: (resource: ontology.Resource) => Haul.Item[];
  ContextMenu?: ContextMenu;
  visible?: (resource: ontology.Resource) => boolean;
  Content?: Content;
}

export const createItem = ({
  type,
  icon,
  useOnSelect = () => noop,
  children,
  hasChildren = children != null,
  canDrop = () => false,
  drop,
  haulItems = () => [],
  ContextMenu,
  visible,
  Content: ContentComp,
}: CreateItemParams): Item => {
  const Row = ContentComp ?? DefaultRow;
  const Component = (props: ItemProps): ReactElement => {
    const onSelect = useOnSelect();
    const handleDoubleClick = () => onSelect(props.resource);
    const resolvedIcon = Icon.resolve(
      typeof icon === "function" ? icon(props.resource) : icon,
    );
    return <Row {...props} icon={resolvedIcon} onDoubleClick={handleDoubleClick} />;
  };
  const item = Component as Item;
  item.type = type;
  item.hasChildren = hasChildren;
  item.children = children;
  item.canDrop = canDrop;
  item.drop = drop;
  item.haulItems = haulItems;
  item.ContextMenu = ContextMenu;
  item.visible = visible;
  return item;
};

// DefaultItem is the fallback used for resource types with no registered Item: an
// unadorned name row with no children and no drag or drop behavior.
export const DefaultItem: Item = createItem({ type: "builtin" });
