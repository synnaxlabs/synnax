// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Size } from "@/component/size";
import { type Select } from "@/select";
import { Buttons } from "@/select/Buttons";
import { Item } from "@/select/Item";

/** Props for {@link SelectSize}. */
export interface SelectComponentSizeProps extends Select.ButtonsProps<Size> {}

/** A button group for picking a {@link Size}, labeled XS through XL. */
export const SelectSize = (props: SelectComponentSizeProps): ReactElement => (
  <Buttons {...props}>
    <Item itemKey="tiny">XS</Item>
    <Item itemKey="small">S</Item>
    <Item itemKey="medium">M</Item>
    <Item itemKey="large">L</Item>
    <Item itemKey="huge">XL</Item>
  </Buttons>
);
