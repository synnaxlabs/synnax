// Copyright 2022 Synnax Labs, Inc.
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
import { InputItem } from "./InputItem";
import { InputLabel } from "./InputLabel";
import { InputSwitch } from "./InputSwitch";
import { InputTime } from "./InputTime";
export type { InputProps } from "./Input";

type CoreInputType = typeof CoreInput;

interface InputType extends CoreInputType {
  Time: typeof InputTime;
  Date: typeof InputDate;
  Label: typeof InputLabel;
  HelpText: typeof InputHelpText;
  Item: typeof InputItem;
  Switch: typeof InputSwitch;
}

export const Input = CoreInput as InputType;

Input.Time = InputTime;
Input.Date = InputDate;
Input.Label = InputLabel;
Input.HelpText = InputHelpText;
Input.Item = InputItem;
Input.Switch = InputSwitch;
