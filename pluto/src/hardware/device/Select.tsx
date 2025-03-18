// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { type AsyncTermSearcher } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Breadcrumb } from "@/breadcrumb";
import { type List } from "@/list";
import { Select } from "@/select";
import { Synnax } from "@/synnax";

const deviceColumns: Array<List.ColumnSpec<device.Key, device.Device>> = [
  { key: "name", name: "Name" },
  {
    key: "location",
    name: "Location",
    render: ({ entry }) => (
      <Breadcrumb.Breadcrumb
        level="small"
        shade={7}
        weight={500}
        style={{ marginTop: "0.25rem" }}
      >
        {entry.location}
      </Breadcrumb.Breadcrumb>
    ),
  },
];

export interface SelectSingleProps
  extends Omit<Select.SingleProps<device.Key, device.Device>, "columns"> {
  searchOptions?: device.RetrieveOptions;
}

export const SelectSingle = ({
  searchOptions,
  ...rest
}: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  let searcher: AsyncTermSearcher<string, device.Key, device.Device> | undefined =
    client?.hardware.devices;
  if (searchOptions != null && client != null)
    searcher = client.hardware.devices.newSearcherWithOptions(searchOptions);
  return (
    <Select.Single<device.Key, device.Device>
      columns={deviceColumns}
      searcher={searcher}
      entryRenderKey="name"
      {...rest}
    />
  );
};
