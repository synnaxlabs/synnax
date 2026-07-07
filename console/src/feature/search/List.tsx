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

import { Palette } from "@/platform/palette";
import { type Search } from "@/platform/search";

const emptyContent = (
  <Text.Text status="disabled" center level="h4">
    <Icon.Resources />
    No resources found
  </Text.Text>
);

export const createList = (items: Search.ListItems) => {
  const filter = (item: ontology.Resource) => items[item.id.type] != null;
  const ListItem = (props: Base.ItemProps<string>) => {
    const item = Base.useItem<string, ontology.Resource>(props.itemKey);
    if (item == null) return null;
    const Item = items[item.id.type];
    if (Item == null) return null;
    return <Item {...props} />;
  };
  const listItem = Component.renderProp(ListItem);
  const List: Palette.List<ontology.Resource> = (props) => {
    const listProps = Ontology.useResourceList({ filter });
    return (
      <Palette.BaseList
        emptyContent={emptyContent}
        listItem={listItem}
        {...listProps}
        {...props}
      />
    );
  };
  List.displayName = "Search.List";
  return List;
};
