import { Form } from "@synnaxlabs/lyra/form";
import { type location } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Label } from "../label";
interface SymbolOrientation {
    label: Label.Config;
    orientation?: location.Outer;
}
interface FieldExtraProps {
    hideOuter?: boolean;
    hideInner?: boolean;
    showOuterCenter?: boolean;
}
export declare const Field: ({ hideOuter, hideInner, showOuterCenter, ...rest }: Form.FieldProps<SymbolOrientation> & FieldExtraProps) => ReactElement | null;
export {};
//# sourceMappingURL=Field.d.ts.map