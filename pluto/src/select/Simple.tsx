import { type record } from "@synnaxlabs/x";

import { Component } from "@/component";
import { Dialog } from "@/dialog";
import { type Icon } from "@/icon";
import { List } from "@/list";
import { type ItemRenderProp } from "@/list/Item";
import { Dialog as SelectDialog } from "@/select/Dialog";
import { Frame, type SingleProps } from "@/select/Frame";
import { SingleTrigger } from "@/select/SingleTrigger";
import { Text } from "@/text";

export interface SimplyEntry<K extends record.Key> extends record.KeyedNamed<K> {
  icon?: Icon.ReactElement;
}

export interface SimpleProps<
  K extends record.Key,
  E extends SimplyEntry<K> = SimplyEntry<K>,
> extends Omit<SingleProps<K, E>, "children"> {
  data: E[];
  children?: ItemRenderProp<K>;
}

const listItem = Component.renderProp((p: List.ItemProps<record.Key>) => {
  const { itemKey } = p;
  const item = List.useItem<record.Key, SimplyEntry<record.Key>>(itemKey);
  if (item == null) return null;
  const { name, icon } = item;
  return (
    <List.Item {...p}>
      <Text.WithIcon level="p" startIcon={icon}>
        {name}
      </Text.WithIcon>
    </List.Item>
  );
});

export const Simple = <K extends record.Key, E extends record.KeyedNamed<K>>({
  data: entries,
  children = listItem,
  emptyContent,
  allowNone,
  value,
  onChange,
  disabled,
  ...rest
}: SimpleProps<K, E>) => {
  const { data, useItem, retrieve } = List.useStaticData<K, E>(entries);
  return (
    <Dialog.Frame {...rest}>
      <Frame<K, E>
        data={data}
        useListItem={useItem}
        value={value}
        onChange={onChange}
        allowNone={allowNone}
      >
        <SingleTrigger disabled={disabled} />
        <SelectDialog onSearch={retrieve} emptyContent={emptyContent}>
          {children}
        </SelectDialog>
      </Frame>
    </Dialog.Frame>
  );
};
