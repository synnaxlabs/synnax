// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/cluster/list/List.css";

import { type connection } from "@synnaxlabs/client";
import { Flex, List, Select, Status, Text, Tooltip } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";

import { useReachability } from "@/platform/cluster/useReachability";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

interface ListItemProps extends List.ItemProps<string> {
  validateName: (name: string) => boolean;
  item: Session.Cluster.Cluster;
  loading: boolean;
}

const LABELS: Record<connection.Status["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  error: "Failed",
  disabled: "Disconnected",
};

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
  const status = useReachability(item);
  let statusVariant = status?.variant ?? "disabled";
  let statusMessage = LABELS[statusVariant];
  if (loading) {
    statusMessage = "Connecting";
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
          {status?.details.nodeVersion != null && (
            <Text.Text size="tiny" color={9}>
              v{status.details.nodeVersion}
            </Text.Text>
          )}
          <Text.Text size="tiny" color={9}>
            {item.host}:{item.port}
          </Text.Text>
        </Flex.Box>
      </Flex.Box>
      <Tooltip.Dialog>
        <Text.Text level="h5">{status?.message}</Text.Text>
        <Status.Summary variant={statusVariant} message={statusMessage} />
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
