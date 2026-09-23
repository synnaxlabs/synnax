import { type text } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";
import { type Component } from "../component";
import { type Theming } from "../theming";
export interface ContextValue {
    onClick: (key: string) => void;
    selected: string;
    level?: text.Level;
    gap?: Component.Size;
    background?: Theming.Shade;
}
declare const useContext: () => ContextValue;
export { useContext };
/** Props for {@link Menu}. */
export interface MenuProps extends PropsWithChildren, Pick<ContextValue, "level" | "gap" | "background"> {
    /** The key of the selected item. */
    value?: string;
    /**
     * Called with the key of the clicked item. Pass a record instead to route each key to
     * its own handler.
     */
    onChange?: ((key: string) => void) | Record<string, (key: string) => void>;
}
/**
 * Holds the shared state for a list of {@link Item}s: which one is selected, what
 * happens on click, and the text level, gap, and background they inherit. It renders no
 * element of its own.
 *
 * @example
 * <Menu.Menu onChange={{ rename, delete: del }}>
 *   <Menu.Item itemKey="rename">Rename</Menu.Item>
 * </Menu.Menu>
 */
export declare const Menu: ({ children, onChange, level, gap, background, value: selected, }: MenuProps) => ReactElement;
//# sourceMappingURL=Menu.d.ts.map