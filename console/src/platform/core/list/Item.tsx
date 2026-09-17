// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/core/list/List.css";

import { type connection } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Icon,
  List,
  Select,
  Status,
  Synnax,
  Text,
  Tooltip,
} from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";

import { Clipboard } from "@/platform/clipboard";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

/** DOM id of a row's editable name, so {@link Text.edit} can target it. */
export const nameID = (key: string): string => `core-${key}`;

interface ListItemProps extends List.ItemProps<string> {
  item: Session.Core.Core;
  loading: boolean;
}

const LABELS: Record<connection.Status["variant"], string> = {
  success: "Connected",
  info: "Connected",
  loading: "Connecting",
  warning: "Reconnecting",
  error: "Unreachable",
  disabled: "Disconnected",
};

const Base = ({ item, loading, ...rest }: ListItemProps): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const copy = Clipboard.useCopy();
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const handleChange = (value: string): boolean => {
    dispatch(Session.Core.rename({ key: item.key, name: value }));
    return true;
  };
  const status = Synnax.useCheckConnection({
    host: item.host,
    port: item.port,
    secure: item.secure,
  });
  let statusVariant = status?.variant ?? "disabled";
  let statusMessage = LABELS[statusVariant];
  if (loading) {
    statusMessage = "Connecting";
    statusVariant = "loading";
  }
  return (
    <List.Item
      className={CSS.cls(CSS.B("core-list-item"))}
      y
      selected={selected}
      onSelect={onSelect}
      gap="small"
      {...rest}
    >
      <Flex.Box x justify="between" align="center" gap="small">
        <Text.MaybeEditable
          id={nameID(item.key)}
          weight={500}
          value={item.name}
          onChange={handleChange}
          allowDoubleClick={false}
          overflow="ellipsis"
          level="h5"
          className={CSS.BE("core-list-item", "name")}
        />
        <Tooltip.Dialog>
          <Text.Text level="h5">{status?.message}</Text.Text>
          <Status.Summary variant={statusVariant} message={statusMessage} />
        </Tooltip.Dialog>
      </Flex.Box>
      <Flex.Box x justify="between" align="center" gap="small">
        <Flex.Box x grow>
          {status?.details.nodeVersion != null && (
            <Text.Text size="tiny" color={9}>
              v{status.details.nodeVersion}
            </Text.Text>
          )}
          <Text.Text size="tiny" color={9} overflow="ellipsis" grow>
            {item.host}:{item.port}
          </Text.Text>
        </Flex.Box>
        <Button.Button
          variant="text"
          size="tiny"
          reveal
          onClick={() => copy(`${item.host}:${item.port}`, "Core address")}
        >
          <Icon.Copy />
        </Button.Button>
      </Flex.Box>
    </List.Item>
  );
};

export const Item = memo((props: Omit<ListItemProps, "item">): ReactElement | null => {
  const item = Session.Core.useSelectState(props.itemKey);
  if (item == null) return null;
  return <Base {...props} item={item} />;
});
Item.displayName = "List.Item";
