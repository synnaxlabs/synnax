// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/cluster/list/List.css";

import {
  Cluster,
  Flex,
  List,
  Select,
  Status,
  Synnax,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Session } from "@/session";

interface ListItemProps extends List.ItemProps<string> {
  validateName: (name: string) => boolean;
  item: Session.Cluster.Cluster;
  loading: boolean;
}

const Base = ({
  validateName,
  item,
  loading,
  ...rest
}: ListItemProps): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const handleChange = (value: string) => {
    if (!validateName(value) || item == null) return;
    dispatch(Session.Cluster.rename({ key: item.key, name: value }));
  };
  const { data } = Cluster.useConnectionState(item);
  const status = data?.status ?? "disconnected";
  let statusVariant = Synnax.CONNECTION_STATE_VARIANTS[status];
  let statusMessage: string = status;
  if (loading) {
    statusMessage = "connecting";
    statusVariant = "loading";
  }
  return (
    <List.Item
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
        <Status.Summary
          variant={statusVariant}
          message={caseconv.capitalize(statusMessage)}
        />
      </Tooltip.Dialog>
    </List.Item>
  );
};

export const Item = memo((props: Omit<ListItemProps, "item">): ReactElement | null => {
  const item = Session.Cluster.useSelectState(props.itemKey);
  if (item == null) return null;
  return <Base {...props} item={item} />;
});
Item.displayName = "List.Item";
