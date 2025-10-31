// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { List as CoreList, Select, Text } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { useSelect } from "@/cluster/selectors";
import { rename } from "@/cluster/slice";
import { CSS } from "@/css";

interface ListItemProps extends CoreList.ItemProps<string> {
  validateName: (name: string) => boolean;
}

const Base = ({ validateName, ...rest }: ListItemProps): ReactElement | null => {
  const dispatch = useDispatch();
  const item = useSelect(rest.itemKey);
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const handleChange = (value: string) => {
    if (!validateName(value) || item == null) return;
    dispatch(rename({ key: item.key, name: value }));
  };
  if (item == null) return null;
  return (
    <CoreList.Item
      className={CSS(CSS.B("cluster-list-item"))}
      y
      selected={selected}
      onSelect={onSelect}
      gap="small"
      {...rest}
    >
      <Text.MaybeEditable
        id={`cluster-dropdown-${item.key}`}
        weight={500}
        value={item.name}
        onChange={handleChange}
        allowDoubleClick={false}
      />
      <Text.Text color={9} weight={450}>
        {item.host}:{item.port}
      </Text.Text>
    </CoreList.Item>
  );
};

export const Item = memo(Base);
