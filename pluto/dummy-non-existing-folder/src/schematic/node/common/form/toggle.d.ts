import { type ReactElement } from "react";
import { type FormProps } from "../../spec";
interface ToggleFormProps extends FormProps {
    hideInnerOrientation?: boolean;
    omit?: string[];
}
export declare const ToggleForm: ({ actions, hideInnerOrientation, omit, }: ToggleFormProps) => ReactElement;
export declare const DummyToggleForm: () => ReactElement;
export {};
//# sourceMappingURL=toggle.d.ts.map