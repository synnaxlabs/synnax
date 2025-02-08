// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { type List } from "@/list";
import { Select } from "@/select";
import { Synnax } from "@/synnax";

const COLUMNS: Array<List.ColumnSpec<rack.RackKey, rack.Rack>> = [
  { key: "name", name: "Name" },
  { key: "location", name: "Location" },
];

export interface SelectSingleProps
  extends Omit<Select.SingleProps<rack.RackKey, rack.Rack>, "columns"> {}

export const SelectSingle = (props: SelectSingleProps): ReactElement => {
  const client = Synnax.use();
  return (
    <Select.Single<rack.RackKey, rack.Rack>
      columns={COLUMNS}
      searcher={client?.hardware.racks}
      entryRenderKey="name"
      {...props}
    />
  );
};
