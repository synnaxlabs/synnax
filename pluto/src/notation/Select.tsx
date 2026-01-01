// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { notation } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Select as CoreSelect } from "@/select";

export interface SelectNotationProps extends Omit<
  CoreSelect.ButtonsProps<notation.Notation>,
  "keys"
> {}

export const Select = (props: SelectNotationProps): ReactElement => (
  <CoreSelect.Buttons {...props} keys={notation.NOTATIONS}>
    <CoreSelect.Button itemKey="standard">Standard</CoreSelect.Button>
    <CoreSelect.Button itemKey="scientific">Scientific</CoreSelect.Button>
    <CoreSelect.Button itemKey="engineering">Engineering</CoreSelect.Button>
  </CoreSelect.Buttons>
);
