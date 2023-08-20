// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { SelectMultiple } from "@/select/Multiple";
import { Select as CoreSelect } from "@/select/Single";

import { SelectButton } from "./Button";
import { SelectDirection } from "./Direction";

export type { SelectMultipleProps } from "@/select/Multiple";
export type { SelectProps } from "@/select/Single";

type CoreSelectType = typeof CoreSelect;

interface SelectType extends CoreSelectType {
  /**
   * Allows a user to browse, search for, and select multiple values from a list of
   * options. It's important to note that Select maintains no internal selection state.
   * The caller must provide the selected value via the `value` prop and handle any
   * changes via the `onChange` prop.
   *
   * @param props - The props for the component. Any additional props will be passed to
   * the input group containing the selection input and the selected tags.
   * @param props.data - The data to be used to populate the select options.
   * @param props.columns - The columns to be used to render the select options in the
   * dropdown. See the {@link ListColumn} type for more details on how to configure
   * columns.
   * @param props.tagKey - The option field rendered for each tag when selected in the
   * input group. Defaults to "key".
   * @param props.location - Whether to render the dropdown above or below the select
   * component. Defaults to "below".
   * @param props.onChange - The callback to be invoked when the selected value changes.
   * @param props.value - The currently selected value.
   */
  Multiple: typeof SelectMultiple;
  Button: typeof SelectButton;
  Direction: typeof SelectDirection;
}

/**
 * Allows a user to browse, search for, and select a value from a list of options.
 * It's important to note that Select maintains no internal selection state. The caller
 * must provide the selected value via the `value` prop and handle any changes via the
 * `onChange` prop.
 *
 * @param props - The props for the component. Any additional props will be passed to the
 * underlying input element.
 * @param props.data - The data to be used to populate the select options.
 * @param props.columns - The columns to be used to render the select options in the
 * dropdown. See the {@link ListColumn} type for more details on available options.
 * @param props.tagKey - The option field rendered when selected. Defaults to "key".
 * @param props.location - Whether to render the dropdown above or below the select
 * component. Defaults to "below".
 * @param props.onChange - The callback to be invoked when the selected value changes.
 * @param props.value - The currently selected value.
 */
export const Select = CoreSelect as SelectType;

Select.Multiple = SelectMultiple;
Select.Button = SelectButton;
Select.Direction = SelectDirection;
