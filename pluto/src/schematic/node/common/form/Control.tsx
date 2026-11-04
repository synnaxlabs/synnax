// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import {
  NegatedSwitchField,
  type NegatedSwitchFieldProps,
} from "@/schematic/node/common/form/NegatedSwitch";

const path = deep.path<{ control: { hidden: boolean } }>();

export const ControlChipField = (
  props: Partial<NegatedSwitchFieldProps>,
): ReactElement => (
  <NegatedSwitchField
    path={path.control.hidden}
    label="Control chip"
    padHelpText={false}
    {...props}
  />
);
