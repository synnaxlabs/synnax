// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/core/list/List.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Tooltip } from "@synnaxlabs/lyra/tooltip";
import { Synnax } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type DragEvent, memo, type ReactElement, useState } from "react";

import { Connection } from "@/platform/connection";
import { CSS } from "@/platform/css";
import { Session } from "@/session";

/** DOM id of a row's editable name, so {@link Text.edit} can target it. */
export const nameID = (key: string): string => `core-${key}`;

interface ListItemProps extends List.ItemProps<string> {
  item: Session.Core.Core;
  loading: boolean;
}

const HAUL_TYPE = "core";
const canDropCore = Haul.canDropOfType(HAUL_TYPE);

const Base = ({ item, loading, ...rest }: ListItemProps): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const { selected, onSelect } = Select.useItemState(rest.itemKey);
  const [over, setOver] = useState(false);
  const { startDrag, onDragEnd, onDragOver, onDrop } = Haul.useDragAndDrop({
    type: HAUL_TYPE,
    key: item.key,
    canDrop: canDropCore,
    onDragOver: () => setOver(true),
    onDrop: ({ items }) => {
      setOver(false);
      const dragged = items.find(({ type }) => type === HAUL_TYPE);
      if (dragged != null && rest.index != null)
        dispatch(Session.Core.reorder({ key: String(dragged.key), index: rest.index }));
      return items;
    },
  });
  const handleDragStart = (event: DragEvent): void =>
    startDrag({
      event,
      items: [{ type: HAUL_TYPE, key: item.key }],
      preview: (
        <Haul.Preview
          icon={<Icon.Core />}
          name={item.name}
          facts={[`${item.host}:${item.port}`]}
        />
      ),
    });
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
  let statusMessage =
    status == null ? Connection.VARIANT_LABELS.disabled : Connection.label(status);
  if (loading) {
    statusVariant = "loading";
    statusMessage = Connection.VARIANT_LABELS.loading;
  }
  return (
    <List.Item
      className={CSS.cls(CSS.B("core-list-item"))}
      y
      selected={selected}
      onSelect={onSelect}
      gap="small"
      draggable
      draggingOver={over}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onDragOver={onDragOver}
      onDragLeave={() => setOver(false)}
      onDrop={onDrop}
      {...rest}
    >
      <Flex.Box x justify="between" align="center" gap="small">
        <Text.MaybeEditable
          id={nameID(item.key)}
          weight={500}
          value={item.name}
          onChange={handleChange}
          allowDoubleClick={false}
          overflow="fade"
          level="h5"
          className={CSS.BE("core-list-item", "name")}
        />
        <Tooltip.Dialog>
          <Flex.Box y gap="tiny">
            <Text.Text level="h5">{status?.message}</Text.Text>
            {primitive.isNonZero(status?.description) && (
              <Text.Text level="small" color={9}>
                {status.description}
              </Text.Text>
            )}
          </Flex.Box>
          <Status.Summary variant={statusVariant} message={statusMessage} />
        </Tooltip.Dialog>
      </Flex.Box>
      <Flex.Box x>
        {status?.details.nodeVersion != null && (
          <Text.Text size="tiny" color={9}>
            v{status.details.nodeVersion}
          </Text.Text>
        )}
        <Text.Text size="tiny" color={9} overflow="fade" grow>
          {item.host}:{item.port}
          {item.user != null && ` · ${item.user}`}
        </Text.Text>
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
