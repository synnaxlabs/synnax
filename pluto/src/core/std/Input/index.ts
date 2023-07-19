// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input as CoreInput } from "@/core/std/Input/Input";
import { InputDate } from "@/core/std/Input/InputDate";
import { InputHelpText } from "@/core/std/Input/InputHelpText";
import { InputItem, InputItemControlled } from "@/core/std/Input/InputItem";
import { InputLabel } from "@/core/std/Input/InputLabel";
import { InputNumber as InputNumeric } from "@/core/std/Input/InputNumber";
import { InputSwitch } from "@/core/std/Input/InputSwitch";
import { InputTime } from "@/core/std/Input/InputTime";
import { combineDatAndTimeValue } from "@/core/std/Input/time";
import { usePassThroughInputControl } from "@/core/std/Input/usePassthroughInputControl";

export type {
  InputControl,
  PartialInputControl,
  InputBaseProps,
} from "@/core/std/Input/types";
export type {
  InputItemProps,
  InputItemControlledProps,
} from "@/core/std/Input/InputItem";
export type { InputProps } from "@/core/std/Input/Input";
export type { InputDateProps } from "@/core/std/Input/InputDate";
export type { InputTimeProps } from "@/core/std/Input/InputTime";
export type { InputLabelProps } from "@/core/std/Input/InputLabel";
export type { InputSwitchProps } from "@/core/std/Input/InputSwitch";
export type { InputHelpTextProps } from "@/core/std/Input/InputHelpText";
export type { InputNumberProps } from "@/core/std/Input/InputNumber";

type CoreInputType = typeof CoreInput;

interface InputType extends CoreInputType {
  /**
   * A controlled Time input component.
   *
   * @param props - The props for the input component. Unlisted props are passed to the
   * underlying input element.
   * @param props.value - The value of the input.
   * @param props.onChange - A function to call when the input value changes.
   * @param props.size - The size of the input: "small" | "medium" | "large".
   * @default "medium"
   * @param props.selectOnFocus - Whether the input should select its contents when focused.
   * @defaul true
   * @param props.centerPlaceholder - Whether the placeholder should be centered.
   * @default false
   * @param props.showDragHandle - Whether or not to show a drag handle to set the time.
   * @default true
   * @param props.dragScale - The scale of the drag handle.
   * @default x: 1/2 Second, y: 1 Minute
   * @param props.dragDirection - The direction of the drag handle.
   * @default undefined
   */
  Time: typeof InputTime;
  /**
   * A controlled Date input component.
   *
   * @param props - The props for the input component. Unlisted props are passed to the
   * underlying input element.
   * @param props.value - The value of the input represented as a number of nanoseconds
   * since the Unix epoch.
   * @param props.onChange - A function to call when the input value changes.
   * @param props.size - The size of the input: "small" | "medium" | "large".
   * @default "medium"
   * @param props.selectOnFocus - Whether the input should select its contents when focused.
   * @defaul true
   * @param props.centerPlaceholder - Whether the placeholder should be centered.
   * @default false
   * @param props.showDragHandle - Whether or not to show a drag handle to set the time.
   * @default true
   * @param props.dragScale - The scale of the drag handle.
   * @default x: 1 Hour, y: 3/4 Day
   * @param props.dragDirection - The direction of the drag handle.
   * @default undefined
   */
  Date: typeof InputDate;
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
  Switch: typeof InputSwitch;
  /**
   * A controlled number input component.
   *
   * @param props - The props for the input component. Unlisted props are passed to the
   * underlying input element.
   * @param props.value - The value of the input.
   * @param props.onChange - A function to call when the input value changes.
   * @param props.size - The size of the input: "small" | "medium" | "large".
   * @default "medium"
   * @param props.selectOnFocus - Whether the input should select its contents when focused.
   * @defaul true
   * @param props.centerPlaceholder - Whether the placeholder should be centered.
   * @default false
   * @param props.showDragHandle - Whether or not to show a drag handle to set the time.
   * @default true
   * @param props.dragScale - The scale of the drag handle.
   * @default x: 1, y: 10
   * @param props.dragDirection - The direction of the drag handle.
   * @default undefined
   */
  Numeric: typeof InputNumeric;
  /**
   * A thin, styled wrapper for an input label. We generally recommend using Input.Item
   * with a 'label' prop instead of this component. This component is useful for
   * low-level control over the label element.
   *
   * @param props - Props for the label component. Unlisted props are passed to the
   * underlying label element.
   */
  Label: typeof InputLabel;
  /**
   * Help text for an input component. We generally recommend using Input.Item with a
   * 'helpText' prop instead of this component. This component is useful for low-level
   * control over the help text element.
   *
   * @param props - Props for the help text component. Unlisted props are passed to the
   * underlying text element.
   * @param props.variant - The variant of the help text.
   * "success" | "error" | "warning" | "info" | "loading" | "disabled
   * @default "info"
   */
  HelpText: typeof InputHelpText;
  Item: typeof InputItem;
  ItemC: typeof InputItemControlled;
  combineDateAndTimeValue: typeof combineDatAndTimeValue;
  usePassthrough: typeof usePassThroughInputControl;
}

/**
 * A controlled string input component.
 *
 * @param props - The props for the input component. Unlisted props are passed to the
 * underlying input element.
 * @param props.value - The value of the input.
 * @param props.onChange - A function to call when the input value changes.
 * @param props.size - The size of the input: "small" | "medium" | "large".
 * @param props.selectOnFocus - Whether the input should select its contents when focused.
 * @param props.centerPlaceholder - Whether the placeholder should be centered.
 */
export const Input = CoreInput as InputType;

Input.Time = InputTime;
Input.Date = InputDate;
Input.Label = InputLabel;
Input.HelpText = InputHelpText;
Input.Item = InputItem;
Input.ItemC = InputItemControlled;
Input.Switch = InputSwitch;
Input.Numeric = InputNumeric;
Input.combineDateAndTimeValue = combineDatAndTimeValue;
Input.usePassthrough = usePassThroughInputControl;
