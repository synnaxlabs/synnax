// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { type Select } from "@/select";
import { Button, Buttons } from "@/select/Button";

export interface AlignmentProps extends Omit<
  Select.ButtonsProps<Flex.Alignment>,
  "keys"
> {}

export const Alignment = ({ value, ...rest }: AlignmentProps): ReactElement => (
  <Buttons {...rest} value={value} keys={Flex.ALIGNMENTS}>
    <Button itemKey="start">
      <Icon.TextAlign.Left />
    </Button>
    <Button itemKey="center">
      <Icon.TextAlign.Center />
    </Button>
    <Button itemKey="end">
      <Icon.TextAlign.Right />
    </Button>
  </Buttons>
);
