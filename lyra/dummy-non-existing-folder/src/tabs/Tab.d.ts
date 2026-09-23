import { type ReactElement } from "react";
import { Button } from "../button";
export interface TabProps extends Omit<Button.ButtonProps<"div">, "el" | "id"> {
    /** itemKey identifies the tab within its Frame's selection and content panels. */
    itemKey: string;
    /** Called when Delete or Backspace is pressed while the tab is focused. */
    onClose?: () => void;
}
/**
 * Tab is a single selectable handle within a Selector. Clicking it (or pressing Enter
 * or Space while it is focused) selects it in the enclosing Frame. Children define its
 * contents: text, an icon, a {@link Close} button, or any combination. When the tab
 * heads an ordered multi-selection (its key is first in the enclosing selection's array
 * value), it is the focused tab and colors itself with the primary theme color.
 */
export declare const Tab: ({ itemKey, className, children, onClick, onKeyDown, onClose, ...rest }: TabProps) => ReactElement;
//# sourceMappingURL=Tab.d.ts.map