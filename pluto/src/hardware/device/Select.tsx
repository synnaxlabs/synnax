// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack } from "@synnaxlabs/client";
import { type AsyncTermSearcher } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useState } from "react";

import { Breadcrumb } from "@/breadcrumb";
import { useAsyncEffect } from "@/hooks";
import { type List } from "@/list";
import { Select } from "@/select";
import { Synnax } from "@/synnax";

interface Entry extends device.Device {
  rackName?: string;
}

const COLUMNS: Array<List.ColumnSpec<device.Key, Entry>> = [
  { key: "name", name: "Name" },
  {
    key: "location",
    name: "Location",
    render: ({ entry: { location, rackName } }) => (
      <Breadcrumb.Breadcrumb
        level="small"
        shade={9}
        weight={450}
        style={{ marginTop: "0.25rem" }}
        size="tiny"
      >
        {rackName ? `${rackName}.${location}` : location}
      </Breadcrumb.Breadcrumb>
    ),
  },
];

export interface SelectSingleProps
  extends Omit<Select.SingleProps<device.Key, Entry>, "columns"> {
  searchOptions?: device.RetrieveOptions;
}

export const SelectSingle = ({
  searchOptions,
  filter: originalFilter,
  ...rest
}: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  let searcher: AsyncTermSearcher<string, device.Key, Entry> | undefined =
    client?.hardware.devices;
  if (searchOptions != null && client != null)
    searcher = client.hardware.devices.newSearcherWithOptions(searchOptions);
  const [rackMap, setRackMap] = useState(new Map<rack.Key, string>());
  const filter = useCallback(
    (items: device.Device[]) => {
      const newItems = originalFilter?.(items) ?? items;
      return newItems.map((item) => ({
        ...item,
        rackName: rackMap.get(item.rack),
      }));
    },
    [originalFilter, rackMap],
  );
  useAsyncEffect(
    async (signal) => {
      if (client == null) {
        setRackMap(new Map());
        return;
      }
      const racks = await client.hardware.racks.retrieve([]);
      if (signal.aborted) return;
      setRackMap(new Map(racks.map(({ key, name }) => [key, name])));
    },
    [client?.key],
  );
  return (
    <Select.Single<device.Key, Entry>
      columns={COLUMNS}
      searcher={searcher}
      entryRenderKey="name"
      filter={filter}
      {...rest}
    />
  );
};
