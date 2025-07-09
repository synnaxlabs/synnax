// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type ReactElement, useState } from "react";

import { Breadcrumb } from "@/breadcrumb";
import { Component } from "@/component";
import { Dialog } from "@/dialog";
import { type ListParams, useList } from "@/hardware/device/queries";
import { Input } from "@/input";
import { List } from "@/list";
import { Select } from "@/select";
import { type state } from "@/state";
import { Text } from "@/text";

const SingleTrigger = (): ReactElement => {
  const [value] = Select.useSelection<device.Key>();
  const item = List.useItem<device.Key, device.Device>(value);
  return (
    <Dialog.Trigger>
      <Text.Text level="p">{item?.name}</Text.Text>
    </Dialog.Trigger>
  );
};

const listItemRenderProp = Component.renderProp(
  ({ itemKey, ...rest }: List.ItemRenderProps<device.Key>) => {
    const item = List.useItem<device.Key, device.Device>(itemKey);
    return (
      <List.Item itemKey={itemKey} {...rest}>
        <Text.Text level="p">{item?.name}</Text.Text>
        <Breadcrumb.Breadcrumb
          level="small"
          shade={9}
          weight={450}
          style={{ marginTop: "0.25rem" }}
          size="tiny"
        >
          {item?.location ?? ""}
        </Breadcrumb.Breadcrumb>
      </List.Item>
    );
  },
);

interface DialogContentProps {
  emptyContent?: ReactElement;
  params?: ListParams;
  retrieve: state.Setter<ListParams>;
}

const DialogContent = ({
  emptyContent,
  retrieve,
  params,
}: DialogContentProps): ReactElement => {
  const [search, setSearch] = useState("");
  return (
    <Dialog.Dialog>
      <Input.Text
        value={search}
        onChange={(v) => {
          setSearch(v);
          retrieve((prev) => ({ ...prev, ...params, search: v }));
        }}
      />
      <List.Items emptyContent={emptyContent}>{listItemRenderProp}</List.Items>
    </Dialog.Dialog>
  );
};

export interface SelectSingleProps extends Select.SingleProps<device.Key> {
  params?: ListParams;
}

export const SelectSingle = ({
  value,
  onChange,
  emptyContent,
  params,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, useListItem, retrieve } = useList();
  const { onSelect, ...selectProps } = Select.useSingle({ value, onChange, data });
  return (
    <Select.Dialog<device.Key, device.Device | undefined>
      value={value}
      onSelect={onSelect}
      useItem={useListItem}
      data={data}
      {...rest}
      {...selectProps}
    >
      <SingleTrigger />
      <DialogContent emptyContent={emptyContent} params={params} retrieve={retrieve} />
    </Select.Dialog>
  );
};
