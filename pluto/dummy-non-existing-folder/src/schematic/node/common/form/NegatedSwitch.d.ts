import { Form } from "@synnaxlabs/lyra/form";
import { type ReactElement } from "react";
export interface NegatedSwitchFieldProps extends Omit<Form.FieldProps<boolean, boolean>, "children"> {
}
/**
 * Binds a switch to a stored boolean that states its non-default condition, such as
 * `fillHidden` or `dblClickNavDisabled`. The switch reads and writes the affirmative,
 * so `label` names what the switch turns on.
 */
export declare const NegatedSwitchField: (props: NegatedSwitchFieldProps) => ReactElement;
//# sourceMappingURL=NegatedSwitch.d.ts.map