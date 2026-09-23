import { Form } from "@synnaxlabs/lyra/form";
import { type ReactElement } from "react";
export interface BoundsFieldsProps extends Pick<Partial<Form.NumericFieldProps>, "hideIfNull" | "padHelpText"> {
    /** Path to the bounds within the symbol's config. */
    path: string;
}
/**
 * The minimum and maximum of a range, each capped by the other so the pair cannot
 * cross.
 */
export declare const BoundsFields: ({ path, ...rest }: BoundsFieldsProps) => ReactElement;
//# sourceMappingURL=Bounds.d.ts.map