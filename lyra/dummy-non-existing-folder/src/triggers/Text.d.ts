import { type ReactElement } from "react";
import { type Generic } from "../generic";
import { Text as Base } from "../text";
import { type Trigger } from "./triggers";
/** Props for {@link Text}. */
export type TextProps<E extends Generic.ElementType = "p"> = Base.TextProps<E> & {
    trigger: Trigger;
};
/** @returns the icons and letters a trigger reads as, per the current platform. */
export declare const toSymbols: (trigger: Trigger) => (ReactElement | string)[];
/**
 * Renders a shortcut as keycaps followed by its label, with the platform's own modifier
 * symbols.
 *
 * @example <Triggers.Text trigger={["Control", "S"]}>Save</Triggers.Text>
 */
export declare const Text: <E extends Generic.ElementType = "p">({ trigger, children, level, ...rest }: TextProps<E>) => ReactElement;
//# sourceMappingURL=Text.d.ts.map