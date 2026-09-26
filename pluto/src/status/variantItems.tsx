// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { Status as Base } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";

const VARIANTS: [Base.Variant, string][] = [
  ["success", "Success"],
  ["error", "Error"],
  ["warning", "Warning"],
  ["info", "Info"],
  ["loading", "Loading"],
  ["disabled", "Disabled"],
];

/** One select item per status variant, each with its indicator and name. */
export const VARIANT_ITEMS: ReactElement[] = VARIANTS.map(([variant, name]) => (
  <Select.Item<Base.Variant> key={variant} itemKey={variant}>
    <Base.Indicator variant={variant} />
    {name}
  </Select.Item>
));
