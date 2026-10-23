// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Flex } from "@/flex";
import { Icon } from "@/icon";
import { type Select } from "@/select";
import { Buttons } from "@/select/Buttons";
import { Item } from "@/select/Item";

export interface AlignmentProps extends Select.ButtonsProps<Flex.Alignment> {}

export const Alignment = ({ value, ...rest }: AlignmentProps): ReactElement => (
  <Buttons {...rest} value={value}>
    <Item itemKey="start">
      <Icon.TextAlign.Left />
    </Item>
    <Item itemKey="center">
      <Icon.TextAlign.Center />
    </Item>
    <Item itemKey="end">
      <Icon.TextAlign.Right />
    </Item>
  </Buttons>
);
