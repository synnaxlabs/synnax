import "./Button.css";
import { TimeSpan } from "@synnaxlabs/x";
import { type KeyboardEventHandler, type ReactElement } from "react";
import { type Generic } from "../generic";
import { type UseHoldProps } from "../hooks";
import { Text } from "../text";
import { Tooltip } from "../tooltip";
import { Triggers } from "../triggers";
/** The elements a Button can render as. `a` makes it a link, `label` a form control. */
export type ElementType = "button" | "a" | "div" | "label" | "textarea";
/** The rest-state emphasis of the button chassis. */
export type Variant = "filled" | "outlined" | "text";
/** The button-specific props {@link ButtonProps} adds to its element's own props. */
export interface ExtensionProps extends Omit<Text.ExtensionProps, "variant">, Tooltip.ExtensionProps {
    /** The rest-state emphasis. Defaults to "outlined". */
    variant?: Variant;
    /** A keyboard trigger that clicks the button while it is mounted. */
    trigger?: Triggers.Trigger;
    /** Renders the trigger's keys beside the label. true shows `trigger`; a trigger of
     * its own shows that instead, for a button whose real shortcut lives elsewhere. */
    triggerIndicator?: boolean | Triggers.Trigger;
    /** Overrides the label color without moving the chassis off its variant. */
    textColor?: Text.TextProps["color"];
    /** The text variant of the label. */
    textVariant?: Text.Variant;
    /** Blocks interaction and dims the button. */
    disabled?: boolean;
    /** Renders the button flat and inert, for a preview of an interface. */
    preview?: boolean;
    /** Swallows the click without calling onClick. Use for a button that is momentarily
     * inapplicable but must not read as disabled. */
    preventClick?: boolean;
    /** Lets the click reach an ancestor's handler. Clicks stop at the button otherwise. */
    propagateClick?: boolean;
    /** Holds onClick until the button has been held this long, filling a progress bar
     * meanwhile. Use it to guard a destructive action. */
    onClickDelay?: number | TimeSpan;
    /** Marks the button as a hidden action its pluto--reveals container shows. */
    reveal?: boolean;
}
/** The props for the {@link Button} component. */
export type ButtonProps<E extends ElementType = "button"> = Omit<Generic.OptionalElementProps<E>, "color" | "onClick" | "onMouseDown" | "onKeyDown" | "onKeyUp"> & ExtensionProps & Pick<UseHoldProps<HTMLElement>, "onClick" | "onMouseDown"> & {
    onKeyDown?: KeyboardEventHandler<HTMLElement>;
    onKeyUp?: KeyboardEventHandler<HTMLElement>;
};
/**
 * The standard clickable. Renders as a `button` unless `el` names another
 * {@link ElementType}, carries an optional keyboard trigger and tooltip, and lays its
 * icons and label out on the shared size scale.
 *
 * @example <Button.Button onClick={save}><Icon.Save />Save</Button.Button>
 * @example <Button.Button variant="text" trigger={["Control", "S"]} triggerIndicator />
 */
export declare const Button: <E extends ElementType = "button">(props: ButtonProps<E>) => ReactElement;
//# sourceMappingURL=Button.d.ts.map