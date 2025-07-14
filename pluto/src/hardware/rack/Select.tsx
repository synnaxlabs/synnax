// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { Dialog } from "@/dialog";
import { Flux } from "@/flux";
import { useList } from "@/hardware/rack/queries";
import { List } from "@/list";
import { type ListParams } from "@/ranger/queries";
import { Select } from "@/select";
import { Text } from "@/text";

export interface SelectSingleProps
  extends Omit<
      Select.SingleFrameProps<rack.Key, rack.Rack | undefined>,
      "data" | "useListItem"
    >,
    Flux.UseListArgs<ListParams, rack.Key, rack.Rack>,
    Omit<Dialog.FrameProps, "onChange">,
    Pick<Select.DialogProps<rack.Key>, "emptyContent"> {}

const listItemRenderProp = Component.renderProp(
  ({ itemKey }: List.ItemRenderProps<rack.Key>) => {
    const item = List.useItem<rack.Key, rack.Rack>(itemKey);
    return <Text.Text level="p">{item?.name}</Text.Text>;
  },
);

export const SelectSingle = ({
  value,
  onChange,
  filter,
  allowNone,
  emptyContent,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, retrieve, getItem, subscribe } = useList();
  const { onFetchMore, onSearch } = Flux.usePager({ retrieve });
  return (
    <Dialog.Frame {...rest}>
      <Select.Frame<rack.Key, rack.Rack | undefined>
        value={value}
        onChange={onChange}
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        allowNone={allowNone}
        onFetchMore={onFetchMore}
      >
        <Select.SingleTrigger />
        <Select.Dialog<rack.Key>
          onSearch={onSearch}
          searchPlaceholder="Search Racks..."
          emptyContent={emptyContent}
        >
          {listItemRenderProp}
        </Select.Dialog>
      </Select.Frame>
    </Dialog.Frame>
  );
};
