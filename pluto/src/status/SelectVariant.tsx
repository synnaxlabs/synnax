// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { type Status as Base } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";

import { VARIANT_ITEMS } from "@/status/variantItems";

export interface SelectVariantProps extends Omit<
  Select.SingleSimpleProps<Base.Variant>,
  "children" | "resourceName"
> {}

export const SelectVariant = (props: SelectVariantProps): ReactElement => (
  <Select.Simple<Base.Variant> {...props} resourceName="variant">
    {VARIANT_ITEMS}
  </Select.Simple>
);
