// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ni } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Device as PDevice, Flux } from "@synnaxlabs/pluto";
import type { ReactElement } from "react";

import type { Device } from "@/platform/device";

const ListItem = (props: List.ItemRenderProps<ni.device.Key>) => {
  const { itemKey } = props;
  const item = List.useItem<ni.device.Key, ni.device.Device>(itemKey);
  if (item == null) return null;
  const { name, model, status } = item;
  return (
    <Select.Item {...props} align="center">
      <Flex.Box y grow>
        <Text.Text el="span" overflow="fade">
          {name}
        </Text.Text>
        <Text.Text el="span" level="small" color={9} overflow="fade">
          {model}
        </Text.Text>
      </Flex.Box>
      <PDevice.StatusIndicator status={status} tooltipLocation="left" />
    </Select.Item>
  );
};

// Matches the item height of the generic device select.
const ITEM_HEIGHT = 48;

const BASE_QUERY = { includeStatus: true } as const;

const SelectList = ({
  value,
  onChange,
  resourceName = "NI device",
  ...rest
}: Device.SelectListProps): ReactElement => {
  const { data, getItem, subscribe, status, more, search } = Flux.useList<
    ni.device.RetrieveRequest,
    ni.device.Device,
    ni.device.RetrieveSingleParams
  >((client) => client.ni.devices, {
    name: "NI devices",
    query: BASE_QUERY,
    itemQuery: (key: ni.device.Key) => ({ ...BASE_QUERY, key }),
  });
  return (
    <Select.Single<ni.device.Key, ni.device.Device | undefined>
      {...rest}
      resourceName={resourceName}
      value={value}
      onChange={onChange}
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      onSearch={search}
      status={status}
      itemHeight={ITEM_HEIGHT}
    >
      {({ key, ...p }) => <ListItem key={key} {...p} />}
    </Select.Single>
  );
};

/** Lists and reads NI devices from the typed device table. */
export const TYPED_SOURCE: Device.SelectSource = {
  List: SelectList,
  retrieve: async (client, key) => {
    const { rack, identifier } = await client.ni.devices.retrieve({ key });
    return { rack, configured: identifier !== "" };
  },
};
