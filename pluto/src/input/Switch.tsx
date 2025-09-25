// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Switch.css";

import { type ReactElement } from "react";

import { Boolean, type BooleanProps } from "@/input/Boolean";

export interface SwitchProps extends Omit<BooleanProps, "inputType"> {}

/**
 * A controlled boolean Switch input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @default "medium"
 */
export const Switch = (props: SwitchProps): ReactElement => (
  <Boolean {...props} inputType="switch" />
);
