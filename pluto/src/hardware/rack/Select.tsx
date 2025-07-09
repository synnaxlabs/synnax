// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { type ReactElement, useState } from "react";

import { Component } from "@/component";
import { Dialog } from "@/dialog";
import { useList } from "@/hardware/rack/queries";
import { Input } from "@/input";
import { List } from "@/list";
import { type ListParams } from "@/ranger/queries";
import { Select } from "@/select";
import { type state } from "@/state";
import { Text } from "@/text";

export interface SelectSingleProps extends Select.SingleProps<rack.Key> {}

const SingleTrigger = (): ReactElement => {
  const [value] = Select.useSelection<rack.Key>();
  const item = List.useItem<rack.Key, rack.Rack>(value);
  return (
    <Dialog.Trigger>
      <Text.Text level="p">{item?.name}</Text.Text>
    </Dialog.Trigger>
  );
};

const listItemRenderProp = Component.renderProp(
  ({ itemKey }: List.ItemRenderProps<rack.Key>) => {
    const item = List.useItem<rack.Key, rack.Rack>(itemKey);
    return <Text.Text level="p">{item?.name}</Text.Text>;
  },
);

interface DialogContentProps {
  retrieve: state.Setter<ListParams>;
}

const DialogContent = ({ retrieve }: DialogContentProps): ReactElement => {
  const [search, setSearch] = useState("");
  return (
    <Dialog.Dialog>
      <Input.Text
        value={search}
        onChange={(v) => {
          setSearch(v);
          retrieve((prev) => ({ ...prev, search: v }));
        }}
      />
      <List.Items>{listItemRenderProp}</List.Items>
    </Dialog.Dialog>
  );
};
export const SelectSingle = ({
  value,
  onChange,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  const { onSelect, ...selectProps } = Select.useSingle({ value, onChange, data });
  return (
    <Select.Dialog<rack.Key, rack.Rack | undefined>
      value={value}
      onSelect={onSelect}
      useItem={useListItem}
      data={data}
      {...rest}
      {...selectProps}
    >
      <SingleTrigger />
      <DialogContent retrieve={retrieve} />
    </Select.Dialog>
  );
};
