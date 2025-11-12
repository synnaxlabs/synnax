// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/cluster/list/List.css";

import {
  Cluster as PCluster,
  Flex,
  List as CoreList,
  Select,
  Status,
  Synnax,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { useSelect } from "@/cluster/selectors";
import { type Cluster, rename } from "@/cluster/slice";
import { CSS } from "@/css";

interface ListItemProps extends CoreList.ItemProps<string> {
  validateName: (name: string) => boolean;
  item: Cluster;
}

const Base = ({ validateName, item, ...rest }: ListItemProps): ReactElement | null => {
  const dispatch = useDispatch();
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const handleChange = (value: string) => {
    if (!validateName(value) || item == null) return;
    dispatch(rename({ key: item.key, name: value }));
  };
  const { data } = PCluster.useConnectionState(item);
  const status = data?.status ?? "disconnected";
  return (
    <CoreList.Item
      className={CSS(CSS.B("cluster-list-item"))}
      x
      selected={selected}
      onSelect={onSelect}
      gap="small"
      justify="between"
      {...rest}
    >
      <Flex.Box y>
        <Text.MaybeEditable
          id={`cluster-dropdown-${item.key}`}
          weight={500}
          value={item.name}
          onChange={handleChange}
          allowDoubleClick={false}
          overflow="ellipsis"
          level="h5"
          className={CSS.BE("cluster-list-item", "name")}
        />
        <Flex.Box x>
          {data?.nodeVersion != null && (
            <Text.Text size="tiny" color={9}>
              v{data.nodeVersion}
            </Text.Text>
          )}
          <Text.Text size="tiny" color={9}>
            {item.host}:{item.port}
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      <Tooltip.Dialog>
        <Text.Text level="h5">{data?.message}</Text.Text>
        <Text.Text size="small" status={Synnax.CONNECTION_STATE_VARIANTS[status]}>
          <Status.Indicator />
          {caseconv.capitalize(status)}
        </Text.Text>
      </Tooltip.Dialog>
    </CoreList.Item>
  );
};

const Wrapper = (props: Omit<ListItemProps, "item">): ReactElement | null => {
  const item = useSelect(props.itemKey);
  if (item == null) return null;
  return <Base {...props} item={item} />;
};

export const Item = memo(Wrapper);
