export type { InputProps } from "./Input";
import { Input as CoreInput } from "./Input";
import { InputDate } from "./InputDate";
import { InputHelpText } from "./InputHelpText";
import { InputItem } from "./InputItem";
import { InputLabel } from "./InputLabel";
import { InputTime } from "./InputTime";

type CoreInputType = typeof CoreInput;

interface InputType extends CoreInputType {
  Time: typeof InputTime;
  Date: typeof InputDate;
  Label: typeof InputLabel;
  HelpText: typeof InputHelpText;
  Item: typeof InputItem;
}

export const Input = CoreInput as InputType;

Input.Time = InputTime;
Input.Date = InputDate;
Input.Label = InputLabel;
Input.HelpText = InputHelpText;
Input.Item = InputItem;
