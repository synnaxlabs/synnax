// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";

const INPUT_PROPS = {
  tooltip: "Write NaN for bad float values instead of skipping the sample",
} as const;

const path = deep.path<{ config: { nanFillEnabled: boolean } }>();

export const NaNFill = () => (
  <Form.CheckboxField
    label="NaN fill"
    path={path.config.nanFillEnabled}
    inputProps={INPUT_PROPS}
  />
);
