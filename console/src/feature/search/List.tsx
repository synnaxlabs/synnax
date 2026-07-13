// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Component, Icon, List as Base, Ontology, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useMemo } from "react";

import { Palette } from "@/platform/palette";
import { type Search } from "@/platform/search";

const emptyContent = (
  <Text.Text status="disabled" center level="h4">
    <Icon.Resources />
    No resources found
  </Text.Text>
);

export interface ListProps extends Palette.ListProps<Tree.Entry> {
  items: Search.ListItems;
}

export const List = ({ items, ...rest }: ListProps): ReactElement => {
  const filter = useCallback(
    (item: Tree.Entry) => items[item.id.type] != null,
    [items],
  );
  const listItem = useMemo(() => {
    const ListItem = (props: Base.ItemProps<string>) => {
      const item = Base.useItem<string, Tree.Entry>(props.itemKey);
      if (item == null) return null;
      const Item = items[item.id.type];
      if (Item == null) return null;
      return <Item {...props} />;
    };
    return Component.renderProp(ListItem);
  }, [items]);
  const listProps = Ontology.useResourceList({
    filter,
    initialQuery: { excludeFieldData: true },
  });
  return (
    <Palette.BaseList
      emptyContent={emptyContent}
      listItem={listItem}
      {...listProps}
      {...rest}
    />
  );
};
List.displayName = "Search.List";
