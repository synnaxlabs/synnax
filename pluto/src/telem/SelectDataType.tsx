// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, DataType, type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { renderProp } from "@/component/renderProp";
import { Dialog } from "@/dialog";
import { List } from "@/list";
import { Select } from "@/select";
import { Text } from "@/text";

const ALL_CAPS = new Set([DataType.UUID, DataType.JSON]);

const DATA: record.KeyedNamed[] = DataType.ALL.filter(
  (d) => d !== DataType.UNKNOWN,
).map((d) => ({
  key: d.toString(),
  name: ALL_CAPS.has(d)
    ? d.toString().toUpperCase()
    : caseconv.capitalize(d.toString()),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new DataType(d.key).isVariable);

export interface DataTypeProps extends Select.SingleProps<string> {
  hideVariableDensity?: boolean;
}

const itemRenderProp = renderProp(
  ({ key, ...rest }: List.ItemRenderProps<string>): ReactElement => {
    const item = List.useItem<string, record.KeyedNamed>(key);
    return (
      <List.Item key={key} itemKey={key} {...rest}>
        <Text.Text level="p">{item?.name}</Text.Text>
      </List.Item>
    );
  },
);

export const SelectDataType = ({
  hideVariableDensity = false,
  value,
  onChange,
  ...rest
}: DataTypeProps): ReactElement => {
  const { useItem, data } = List.useStaticData<string, record.KeyedNamed>(
    hideVariableDensity ? FIXED_DENSITY_DATA : DATA,
  );
  const { onSelect } = Select.useSingle({ data, value, onChange });
  const selected = useItem(value);
  return (
    <Select.Dialog
      {...rest}
      onSelect={onSelect}
      value={value}
      useItem={useItem}
      data={data}
    >
      <Dialog.Trigger variant="outlined">{selected?.name}</Dialog.Trigger>
      <List.Items>{itemRenderProp}</List.Items>
    </Select.Dialog>
  );
};
