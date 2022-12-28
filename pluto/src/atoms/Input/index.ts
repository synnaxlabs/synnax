import { Input as CoreInput } from "./Input";
import { InputDate } from "./InputDate";
import { InputHelpText } from "./InputHelpText";
import { InputItemControlled, InputItem } from "./InputItem";
import { InputLabel } from "./InputLabel";
import { InputSwitch } from "./InputSwitch";
import { InputTime } from "./InputTime";
export type { InputItemProps } from "./InputItem";
export type { InputProps } from "./Input";
export type { InputDateProps } from "./InputDate";
export type { InputTimeProps } from "./InputTime";
export type { InputLabelProps } from "./InputLabel";
export type { InputSwitchProps } from "./InputSwitch";
export type { InputHelpTextProps } from "./InputHelpText";
export type { InputItemControlledProps } from "./InputItem";
export type { InputControlProps, InputBaseProps } from "./types";

type CoreInputType = typeof CoreInput;

interface InputType extends CoreInputType {
  Time: typeof InputTime;
  Date: typeof InputDate;
  Label: typeof InputLabel;
  HelpText: typeof InputHelpText;
  Item: typeof InputItem;
  ItemC: typeof InputItemControlled;
  Switch: typeof InputSwitch;
}

export const Input = CoreInput as InputType;

Input.Time = InputTime;
Input.Date = InputDate;
Input.Label = InputLabel;
Input.HelpText = InputHelpText;
Input.Item = InputItem;
Input.Switch = InputSwitch;
Input.ItemC = InputItemControlled;
