// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Checkbox.css";

import { type ReactElement } from "react";

import { type Button } from "@/button";
import { Boolean } from "@/input/Boolean";
import { type InputProps } from "@/input/types";

export interface CheckboxProps
  extends InputProps<boolean>,
    Omit<Button.ExtensionProps, "variant"> {}

/**
 * A controlled boolean Checkbox input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 */
export const Checkbox = (props: CheckboxProps): ReactElement => (
  <Boolean inputType="checkbox" {...props} />
);
