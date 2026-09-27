// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Select } from "@/select";
import { Buttons } from "@/select/Buttons";
import { Item } from "@/select/Item";

export interface WeightProps extends Select.ButtonsProps<text.Weight> {}

export const Weight = (props: WeightProps): ReactElement => (
  <Buttons {...props}>
    <Item itemKey={250}>Light</Item>
    <Item itemKey={400}>Normal</Item>
    <Item itemKey={500}>Medium</Item>
    <Item itemKey={600}>Bold</Item>
  </Buttons>
);
