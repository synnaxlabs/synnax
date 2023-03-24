// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Input as CoreInput } from "./Input";
import { InputDate } from "./InputDate";
import { InputHelpText } from "./InputHelpText";
import { InputItem, InputItemControlled } from "./InputItem";
import { InputLabel } from "./InputLabel";
import { InputNumber } from "./InputNumber";
import { InputSwitch } from "./InputSwitch";
import { InputTime } from "./InputTime";
import { parseDateAndTimeInput } from "./time";
export type { InputControl, InputBaseProps } from "./types";
export type { InputItemProps, InputItemControlledProps } from "./InputItem";
export type { InputProps } from "./Input";
export type { InputDateProps } from "./InputDate";
export type { InputTimeProps } from "./InputTime";
export type { InputLabelProps } from "./InputLabel";
export type { InputSwitchProps } from "./InputSwitch";
export type { InputHelpTextProps } from "./InputHelpText";
export type { InputNumberProps } from "./InputNumber";

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
