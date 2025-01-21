// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device as Device } from "@synnaxlabs/client";
import { type AsyncTermSearcher } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type List } from "@/list";
import { Select } from "@/select";
import { Synnax } from "@/synnax";

const deviceColumns: Array<List.ColumnSpec<Device.Key, Device.Device>> = [
  { key: "name", name: "Name" },
  { key: "location", name: "Location" },
];

export interface SelectSingleProps
  extends Omit<Select.SingleProps<Device.Key, Device.Device>, "columns"> {
  searchOptions?: Device.RetrieveOptions;
}

export const SelectSingle = ({
  searchOptions,
  ...props
}: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  let searcher: AsyncTermSearcher<string, Device.Key, Device.Device> | undefined =
    client?.hardware.devices;
  if (searchOptions != null && client != null)
    searcher = client.hardware.devices.newSearcherWithOptions(searchOptions);
  return (
    <Select.Single<Device.Key, Device.Device>
      columns={deviceColumns}
      searcher={searcher}
      entryRenderKey={"name"}
      {...props}
    />
  );
};
