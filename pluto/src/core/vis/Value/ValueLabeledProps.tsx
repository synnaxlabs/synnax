// Copyrght 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { z } from "zod";

import { ValueCore, ValueCoreProps, valueCoreProps } from "./ValueCore";

import { Pack, PackProps, TextMaybeEditableProps } from "@/core/std";
import { TextMaybeEditable } from "@/core/std/Typography/TextEditable";

export const valueLabeledProps = valueCoreProps
  .extend({
    value: z.string(),
  })
  .partial({ box: true });

export interface ValueLabeledProps
  extends ValueCoreProps,
    Omit<PackProps, "color" | "onChange">,
    Pick<TextMaybeEditableProps, "value" | "onChange"> {}

export const ValueLabeled = ({
  onChange,
  value,
  level = "p",
  color,
  ...props
}: ValueLabeledProps): ReactElement => (
  <Pack {...props}>
    <TextMaybeEditable value={value} onChange={onChange} level={level} />
    <ValueCore color={color} level={level} {...props} />
  </Pack>
);
