import { type ComponentPropsWithRef } from "react";
import { type Button } from "../button";
/**
 * The controlled-input contract. Every input in the library takes it, so a form field
 * binds to any of them the same way.
 */
export interface Control<I = unknown, O = I> {
    value: I;
    onChange: (value: O) => void;
}
/** A {@link Control} whose value and handler may both be absent. */
export interface OptionalControl<I = unknown, O = I> extends Partial<Control<I, O>> {
}
/** Native input props an input forwards to its element. */
export type HTMLInputProps = Omit<ComponentPropsWithRef<"input">, "size" | "onChange" | "value" | "children" | "placeholder" | "color">;
export type Variant = Extract<Button.Variant, "outlined" | "text"> | "shadow";
/** Base props shared by every input in the library. */
export interface InputProps<I = unknown, O = I> extends HTMLInputProps, Control<I, O>, Pick<Button.ExtensionProps, "preview"> {
    variant?: Variant;
}
//# sourceMappingURL=types.d.ts.map