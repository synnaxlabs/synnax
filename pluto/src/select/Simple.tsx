import { type record } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { Component } from "@/component";
import { Dialog } from "@/dialog";
import { Input } from "@/input";
import { List } from "@/list";
import { type ItemRenderProp } from "@/list/Item";
import { Dialog as SelectDialog, type SingleProps } from "@/select/Dialog";
import { useSingle } from "@/select/use";
import { Text } from "@/text";

export interface SimpleProps<K extends record.Key, E extends record.KeyedNamed<K>>
  extends SingleProps<K> {
  data: E[];
  children?: ItemRenderProp<K>;
}

interface SearchInputProps {
  onSearch: (term: string) => void;
}

const SearchInput = ({ onSearch }: SearchInputProps) => {
  const [search, setSearch] = useState("");
  const handleChange = useCallback(
    (v: string) => {
      setSearch(v);
      onSearch(v);
    },
    [onSearch],
  );
  return <Input.Text value={search} onChange={handleChange} />;
};

const listItem = Component.renderProp((p: List.ItemProps<record.Key>) => {
  const { itemKey } = p;
  const item = List.useItem<record.Key, record.KeyedNamed<record.Key>>(itemKey);
  if (item == null) return null;
  const { name } = item;
  return (
    <List.Item {...p}>
      <Text.Text level="p">{name}</Text.Text>
    </List.Item>
  );
});

export const Simple = <K extends record.Key, E extends record.KeyedNamed<K>>({
  data: entries,
  value,
  onChange,
  allowNone,
  children = listItem,
  ...rest
}: SimpleProps<K, E>) => {
  const { data, useItem, retrieve } = List.useStaticData<K, E>(entries);
  const selectProps = useSingle<K>({ data, value, onChange });
  const selected = useItem(value);
  const handleSearch = useCallback((term: string) => retrieve({ term }), [retrieve]);
  return (
    <SelectDialog {...rest} {...selectProps} data={data} useItem={useItem}>
      <Dialog.Trigger variant="outlined">{selected?.name ?? "Select"}</Dialog.Trigger>
      <Dialog.Dialog>
        <SearchInput onSearch={handleSearch} />
        <List.Items>{children}</List.Items>
      </Dialog.Dialog>
    </SelectDialog>
  );
};
