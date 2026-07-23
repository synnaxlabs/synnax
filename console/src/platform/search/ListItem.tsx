// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type Icon, List, Text } from "@synnaxlabs/pluto";
import { type FC, useCallback } from "react";

import { Palette } from "@/platform/palette";
import { Panel } from "@/platform/panel";

interface BaseListItemProps extends Omit<Palette.ListItemProps, "onSelect"> {
  icon: Icon.ReactElement;
  onSelect: (item: ontology.Resource) => void;
}

export const BaseListItem = ({ icon, onSelect, ...rest }: BaseListItemProps) => {
  const { itemKey } = rest;
  const item = List.useItem<string, ontology.Resource>(itemKey);
  if (item == null) return null;
  const { name } = item;
  const handleSelect = useCallback(() => onSelect(item), [onSelect, item]);
  return (
    <Palette.ListItem {...rest} onSelect={handleSelect}>
      <Text.Text weight={450} gap="medium">
        {icon}
        {name}
      </Text.Text>
    </Palette.ListItem>
  );
};

export interface ListItemProps extends Omit<BaseListItemProps, "icon" | "onSelect"> {}

export interface ListItem extends FC<ListItemProps> {}

export interface ListItems extends Partial<Record<ontology.ResourceType, ListItem>> {}

export interface CreateListItemParams extends Pick<BaseListItemProps, "icon"> {
  useOnSelect?: () => (item: ontology.Resource) => void;
}

export const createListItem = ({
  icon,
  useOnSelect = Panel.useOpenResource,
}: CreateListItemParams): ListItem => {
  const Item = (props: ListItemProps) => (
    <BaseListItem icon={icon} onSelect={useOnSelect()} {...props} />
  );
  return Item;
};
