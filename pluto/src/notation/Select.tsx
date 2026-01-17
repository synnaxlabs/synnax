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

import { Select as BaseSelect } from "@/select";

export interface SelectNotationProps extends Omit<
  BaseSelect.ButtonsProps<notation.Notation>,
  "keys"
> {}

export const Select = (props: SelectNotationProps): ReactElement => (
  <BaseSelect.Buttons {...props} keys={notation.NOTATIONS}>
    <BaseSelect.Button itemKey="standard">Standard</BaseSelect.Button>
    <BaseSelect.Button itemKey="scientific">Scientific</BaseSelect.Button>
    <BaseSelect.Button itemKey="engineering">Engineering</BaseSelect.Button>
  </BaseSelect.Buttons>
);
