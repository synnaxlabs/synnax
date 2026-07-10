// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { type Haul, Icon, List, Text, Tree as Base } from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useCallback } from "react";

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

// Item renders a single resource row in the ontology tree. The component carries the
// static, render-independent behavior for its resource type (children, drag, context
// menu, visibility) as properties, read imperatively by the Tree.
export interface Item extends FC<ItemProps> {
  type: ontology.ResourceType;
  hasChildren: boolean;
  canDrop: Haul.CanDrop;
  haulItems: (resource: ontology.Resource) => Haul.Item[];
  ContextMenu?: ContextMenu;
  visible?: (resource: ontology.Resource) => boolean;
}

export interface Items extends Partial<Record<ontology.ResourceType, Item>> {}

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
      id={List.itemNameID(ontology.idToString(resource.id))}
      value={resource.name}
      onChange
      allowDoubleClick={false}
      style={{ userSelect: "none", width: 0, flexGrow: 1 }}
      overflow="ellipsis"
    />
  </Base.Item>
);

const noop = (): void => {};

export interface CreateItemArgs {
  type: ontology.ResourceType;
  icon?: Icon.ReactElement | ((resource: ontology.Resource) => Icon.ReactElement);
  useOnSelect?: () => (resource: ontology.Resource) => void;
  hasChildren?: boolean;
  canDrop?: Haul.CanDrop;
  haulItems?: (resource: ontology.Resource) => Haul.Item[];
  ContextMenu?: ContextMenu;
  visible?: (resource: ontology.Resource) => boolean;
  Content?: Content;
}

export const createItem = ({
  type,
  icon,
  useOnSelect = () => noop,
  hasChildren = true,
  canDrop = () => false,
  haulItems = () => [],
  ContextMenu,
  visible,
  Content: ContentComp,
}: CreateItemArgs): Item => {
  const Row = ContentComp ?? DefaultRow;
  const Component = (props: ItemProps): ReactElement => {
    const onSelect = useOnSelect();
    const handleDoubleClick = useCallback(
      () => onSelect(props.resource),
      [onSelect, props.resource],
    );
    const resolvedIcon = Icon.resolve(
      typeof icon === "function" ? icon(props.resource) : icon,
    );
    return <Row {...props} icon={resolvedIcon} onDoubleClick={handleDoubleClick} />;
  };
  const item = Component as Item;
  item.type = type;
  item.hasChildren = hasChildren;
  item.canDrop = canDrop;
  item.haulItems = haulItems;
  item.ContextMenu = ContextMenu;
  item.visible = visible;
  return item;
};

// DefaultItem is the fallback used for resource types with no registered Item: an
// unadorned name row with expandable children and no drag or drop behavior.
export const DefaultItem: Item = createItem({ type: "builtin" });
