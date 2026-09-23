import { type ReactElement } from "react";
import { Icon } from "../icon";
import { type Variant } from "./status";
/** Props for {@link Indicator}. */
export interface IndicatorProps extends Icon.IconProps {
    variant?: Variant;
    /** Replaces the concentric glyph, tinted with the variant color. */
    children?: ReactElement<Icon.IconProps>;
}
/**
 * The dot that carries a status variant's color. A loading variant spins instead, and
 * a child icon takes the color in place of the dot.
 */
export declare const Indicator: ({ variant, children, ...rest }: IndicatorProps) => ReactElement;
//# sourceMappingURL=Indicator.d.ts.map