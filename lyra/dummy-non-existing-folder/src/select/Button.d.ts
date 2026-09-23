import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Button as Base } from "../button";
import { Flex } from "../flex";
import { type FrameProps } from "./Frame";
export interface ButtonsProps<K extends record.Key = record.Key, E extends record.Keyed<K> | undefined = record.Keyed<K>> extends Omit<Flex.BoxProps, "onSelect" | "onChange">, Omit<FrameProps<K, E>, "getItem" | "subscribe" | "data"> {
    /** The selectable keys, in render order. */
    keys: K[] | readonly K[];
    /** Whether to render the buttons flat and inert, for use inside a preview. */
    preview?: boolean;
}
/**
 * A packed row of {@link Button}s acting as one selection. Use it in place of a dropdown
 * when the options are few and fixed.
 *
 * @example
 * <Select.Buttons keys={MODES} value={mode} onChange={setMode}>
 *   <Select.Button itemKey="fast">Fast</Select.Button>
 * </Select.Buttons>
 */
export declare const Buttons: <K extends record.Key = record.Key>({ keys, value, onChange, allowNone, multiple, preview, children, ...rest }: ButtonsProps<K>) => ReactElement;
/** Props for {@link Button}. */
export interface ButtonProps<K extends record.Key = record.Key> extends Omit<Base.ToggleProps, "onChange" | "value"> {
    /** The key this button selects. */
    itemKey: K;
}
/** One option inside {@link Buttons}. It toggles on when its `itemKey` is selected. */
export declare const Button: <K extends record.Key = record.Key>({ itemKey, className, ...rest }: ButtonProps<K>) => ReactElement | null;
//# sourceMappingURL=Button.d.ts.map