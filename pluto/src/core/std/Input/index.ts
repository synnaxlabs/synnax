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
import { InputNumber } from "@/core/std/Input/InputNumber";
import { InputSwitch } from "@/core/std/Input/InputSwitch";
import { InputTime } from "@/core/std/Input/InputTime";
import { parseDateAndTimeInput } from "@/core/std/Input/time";
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
  Time: typeof InputTime;
  Date: typeof InputDate;
  Switch: typeof InputSwitch;
  Number: typeof InputNumber;
  Label: typeof InputLabel;
  HelpText: typeof InputHelpText;
  Item: typeof InputItem;
  ItemC: typeof InputItemControlled;
  combineDateAndTimeValue: typeof parseDateAndTimeInput;
  usePassthrough: typeof usePassThroughInputControl;
}

export const Input = CoreInput as InputType;

Input.Time = InputTime;
Input.Date = InputDate;
Input.Label = InputLabel;
Input.HelpText = InputHelpText;
Input.Item = InputItem;
Input.ItemC = InputItemControlled;
Input.Switch = InputSwitch;
Input.Number = InputNumber;
Input.combineDateAndTimeValue = parseDateAndTimeInput;
Input.usePassthrough = usePassThroughInputControl;
