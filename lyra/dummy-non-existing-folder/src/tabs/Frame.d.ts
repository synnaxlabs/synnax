import "./Tabs.css";
import { type ReactElement } from "react";
import { Flex } from "../flex";
import { state } from "../state";
declare const useFrameID: (hookOrComponentName: string) => string;
export { useFrameID };
/** tabID returns the DOM id of the tab handle for the given key within a Frame. */
export declare const tabID: (frameID: string, key: string) => string;
/** panelID returns the DOM id of the content panel for the given key within a Frame. */
export declare const panelID: (frameID: string, key: string) => string;
/** KEY_ATTRIBUTE is the DOM attribute a Tab renders to expose its key. */
export declare const KEY_ATTRIBUTE = "data-tab-key";
/** KEY_SELECTOR matches the tab handles rendered by Tab via {@link KEY_ATTRIBUTE}. */
export declare const KEY_SELECTOR = "[data-tab-key]";
/**
 * Props for {@link Frame}. Give `value`, `initialValue`, or `onChange` to make the
 * frame own its selection; give none to bind it to an enclosing one.
 */
export interface FrameProps extends Omit<Flex.BoxProps, "onChange" | "onSelect">, Partial<state.UsePurePassthroughProps<string>> {
}
/**
 * Frame is the root of a composed tabbed interface. When given a value, initialValue,
 * or onChange it owns the selected tab key (controlled or uncontrolled) and publishes
 * it to descendants through the Selection context, so only the tabs whose selected
 * state changes re-render. When given no selection at all, the Frame owns nothing: its
 * tabs bind to the nearest enclosing selection context, letting a composite like
 * Panel.Mosaic distribute a single selection across many frames.
 */
export declare const Frame: ({ value, initialValue, onChange, ...rest }: FrameProps) => ReactElement;
//# sourceMappingURL=Frame.d.ts.map