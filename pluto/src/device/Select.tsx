// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/device/Select.css";

import { type device } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Location, type LocationFormat } from "@/device/Location";
import { type RetrieveMultipleQuery, useList } from "@/device/queries";
import { StatusBadge } from "@/device/StatusBadge";
import { StatusIndicator } from "@/device/StatusIndicator";
import { type Flux } from "@/flux";

interface ListItemProps extends List.ItemRenderProps<device.Key> {
  locationFormat?: LocationFormat;
}

const ListItem = ({ itemKey, locationFormat, ...rest }: ListItemProps) => {
  const item = List.useItem<device.Key, device.Device>(itemKey);
  if (item == null) return null;
  const { name, location, status } = item;
  return (
    <Select.Item
      itemKey={itemKey}
      {...rest}
      className={CSS.BE("device", "list-item")}
      align="center"
    >
      <Flex.Box y grow className={CSS.BE("device", "list-item-text")}>
        <Text.Text
          el="span"
          overflow="fade"
          className={CSS.BE("device", "list-item-name")}
        >
          {name}
        </Text.Text>
        {location !== "" && (
          <Location
            value={location}
            format={locationFormat}
            className={CSS.BE("device", "list-item-location")}
          />
        )}
      </Flex.Box>
      <StatusIndicator status={status} tooltipLocation="left" />
    </Select.Item>
  );
};

// Matches the item height in Select.css.
const ITEM_HEIGHT = 48;

const DEVICE_ICON = <Icon.Device />;

const DIALOG_PROPS = { className: CSS.BE("device-select", "dialog") };

const renderStatus = (entry: unknown): ReactElement | undefined =>
  entry == null ? undefined : <StatusBadge status={(entry as device.Device).status} />;

export interface SelectSingleProps
  extends
    Omit<
      Select.SingleProps<device.Key, device.Device | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children" | "itemHeight"
    >,
    Omit<Flux.UseListParams<RetrieveMultipleQuery, device.Device>, "name"> {
  /** How the location of each listed device reads. */
  locationFormat?: LocationFormat;
}

/** Selects one device. The trigger shows the selected device's status. */
export const SelectSingle = ({
  filter,
  query,
  triggerProps,
  icon = DEVICE_ICON,
  locationFormat,
  ...rest
}: SelectSingleProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = useList({
    filter,
    query,
  });
  return (
    <Select.Single<device.Key, device.Device | undefined>
      resourceName="device"
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      onSearch={search}
      status={status}
      icon={icon}
      itemHeight={ITEM_HEIGHT}
      dialogProps={DIALOG_PROPS}
      triggerProps={{ renderEndContent: renderStatus, ...triggerProps }}
      {...rest}
    >
      {({ key, ...p }) => <ListItem key={key} {...p} locationFormat={locationFormat} />}
    </Select.Single>
  );
};
