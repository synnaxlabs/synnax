// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Component } from "@/component";
import { type Select } from "@/select";
import { Buttons } from "@/select/Buttons";
import { Item } from "@/select/Item";

/** Props for {@link Size}. */
export interface SizeProps extends Select.ButtonsProps<Component.Size> {}

/** A button group for picking a component size, labeled XS through XL. */
export const Size = (props: SizeProps): ReactElement => (
  <Buttons {...props}>
    <Item itemKey="tiny">XS</Item>
    <Item itemKey="small">S</Item>
    <Item itemKey="medium">M</Item>
    <Item itemKey="large">L</Item>
    <Item itemKey="huge">XL</Item>
  </Buttons>
);
