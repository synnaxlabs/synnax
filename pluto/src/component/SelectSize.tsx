// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Size, SIZES } from "@/component/size";
import { type Select } from "@/select";
import { Button, Buttons } from "@/select/Button";

/** Props for {@link SelectSize}. */
export interface SelectComponentSizeProps extends Omit<
  Select.ButtonsProps<Size>,
  "keys"
> {}

/** A button group for picking a {@link Size}, labeled S, M, and L. */
export const SelectSize = (props: SelectComponentSizeProps): ReactElement => (
  <Buttons {...props} keys={SIZES}>
    <Button itemKey="tiny">S</Button>
    <Button itemKey="small">M</Button>
    <Button itemKey="medium">L</Button>
  </Buttons>
);
