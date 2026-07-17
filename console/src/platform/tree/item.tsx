// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import {
  type Flux,
  type Haul,
  Icon,
  List,
  Text,
  Tree as Base,
} from "@synnaxlabs/pluto";
import { type FC, type ReactElement, useCallback, useEffect } from "react";

import { type ContextMenu, type Entry } from "@/platform/tree/types";

// UseName reactively resolves the display name for a resource of the Item's type from
// that type's own flux store. Empty string means "not resolved yet".
export interface UseName {
  (id: ontology.ID): string;
}

// ItemProps are the props the Tree passes to a resource type's Item. The Item owns its
// own icon, name resolution, and double-click behavior, so none is threaded through.
export interface ItemProps extends Omit<Base.ItemProps<string>, "id" | "resource"> {
  id: ontology.ID;
  loading: boolean;
  // onName reports the resolved name back to the Tree so it can sort siblings.
  onName?: (name: string) => void;
}

// ContentProps extend ItemProps with the resolved name plus the icon and double-click
// handler the factory resolves. A custom Content receives these; the default row
// renders them directly.
export interface ContentProps extends ItemProps {
  name: string;
  icon?: Icon.ReactElement;
  onDoubleClick: () => void;
}

export interface Content extends FC<ContentProps> {}

// Item renders a single resource row in the ontology tree. The component carries the
// static, render-independent behavior for its resource type (children, drag, context
// menu, visibility) as properties, read imperatively by the Tree.
export interface Item extends FC<ItemProps> {
  type: ontology.ResourceType;
  useName: UseName;
  hasChildren: boolean;
  canDrop: Haul.CanDrop;
  haulItems: (resource: ontology.Resource, store: Flux.Store) => Haul.Item[];
  ContextMenu?: ContextMenu;
  visible?: (id: ontology.ID) => boolean;
}

export interface Items extends Partial<Record<ontology.ResourceType, Item>> {}

export const DefaultRow = ({
  onDoubleClick,
  id: _id,
  name,
  icon,
  loading,
  ...rest
}: ContentProps): ReactElement => (
  <Base.Item {...rest} onDoubleClick={onDoubleClick}>
    {icon}
    <Text.MaybeEditable
      id={List.itemNameID(rest.itemKey)}
      value={name}
      onChange
      allowDoubleClick={false}
      style={{ userSelect: "none", width: 0, flexGrow: 1 }}
      overflow="ellipsis"
    />
  </Base.Item>
);

const noop = (): void => {};
const useEmptyName: UseName = () => "";

export interface CreateItemArgs {
  type: ontology.ResourceType;
  // useName resolves the row's display name from the resource type's flux store. Omit
  // only for types with no name (falls back to empty).
  useName?: UseName;
  icon?: Icon.ReactElement | ((id: ontology.ID) => Icon.ReactElement);
  useOnSelect?: () => (entry: Entry) => void;
  hasChildren?: boolean;
  canDrop?: Haul.CanDrop;
  haulItems?: (resource: ontology.Resource, store: Flux.Store) => Haul.Item[];
  ContextMenu?: ContextMenu;
  visible?: (id: ontology.ID) => boolean;
  Content?: Content;
}

export const createItem = ({
  type,
  useName = useEmptyName,
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
    const { id, itemKey, onName } = props;
    const name = useName(id);
    useEffect(() => onName?.(name), [name, onName]);
    const handleDoubleClick = useCallback(
      () => onSelect({ key: itemKey, id, name }),
      [onSelect, itemKey, id, name],
    );
    const resolvedIcon = Icon.resolve(typeof icon === "function" ? icon(id) : icon);
    return (
      <Row
        {...props}
        name={name}
        icon={resolvedIcon}
        onDoubleClick={handleDoubleClick}
      />
    );
  };
  const item = Component as Item;
  item.type = type;
  item.useName = useName;
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
