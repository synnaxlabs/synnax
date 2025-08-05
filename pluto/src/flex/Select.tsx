// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Alignment, ALIGNMENTS } from "@/flex/Box";
import { Icon } from "@/icon";
import { Select as Core } from "@/select";

export interface SelectAlignmentProps
  extends Omit<Core.ButtonsProps<Alignment>, "keys"> {}

export const SelectAlignment = ({
  value,
  ...rest
}: SelectAlignmentProps): ReactElement => (
  <Core.Buttons {...rest} value={value} keys={ALIGNMENTS}>
    <Core.Button itemKey="start">
      <Icon.TextAlign.Left />
    </Core.Button>
    <Core.Button itemKey="center">
      <Icon.TextAlign.Center />
    </Core.Button>
    <Core.Button itemKey="end">
      <Icon.TextAlign.Right />
    </Core.Button>
  </Core.Buttons>
);
