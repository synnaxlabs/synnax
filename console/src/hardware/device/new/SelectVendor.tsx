// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button, Select } from "@synnaxlabs/pluto";
import { type List } from "@synnaxlabs/pluto/list";

import { type Vendor } from "./types";

interface VendorListItem {
  key: Vendor;
  logo: ReactElement;
  name: string;
}

const DEVICE_VENDORS: VendorListItem[] = [
  {
    key: "ni",
    logo: <Icon.Logo.NI className="vendor-logo " />,
    name: "National Instruments",
  },
  {
    key: "other",
    logo: <Icon.Hardware className="vendor-logo " />,
    name: "Other",
  },
];

const COLUMNS: Array<List.ColumnSpec<Vendor, VendorListItem>> = [
  {
    key: "logo",
    render: ({ entry }) => entry.logo,
    name: "",
    visible: true,
  },
  {
    key: "name",
    name: "Vendor",
    visible: true,
  },
];

export interface SelectVendorProps
  extends Omit<
    Select.DropdownButtonProps<Vendor, VendorListItem>,
    "data" | "columns"
  > {}

export const SelectVendor = (props: SelectVendorProps): ReactElement => (
  <Select.DropdownButton<Vendor, VendorListItem>
    data={DEVICE_VENDORS}
    columns={COLUMNS}
    {...props}
  >
    {({ selected: s, toggle }) => (
      <Select.BaseButton
        iconSpacing="small"
        onClick={toggle}
        variant="outlined"
        startIcon={s?.logo}
      >
        {s?.name}
      </Select.BaseButton>
    )}
  </Select.DropdownButton>
);
