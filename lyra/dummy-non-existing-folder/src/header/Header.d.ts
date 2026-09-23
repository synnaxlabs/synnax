import "./Header.css";
import { type text } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";
import { Flex } from "../flex";
/** Props for {@link Header}. */
export interface HeaderProps extends Omit<Flex.BoxProps, "children" | "el"> {
    /** Type scale step for the title and its actions. Defaults to "h1". */
    level?: text.Level;
    /** Whether to draw a rule between the title and the actions. */
    divided?: boolean;
    bordered?: boolean;
    children: ReactNode | [ReactNode, ReactNode];
}
export interface ContextValue {
    divided: boolean;
    level: text.Level;
}
declare const useContext: () => ContextValue;
export { useContext };
/**
 * The bar at the top of a module. It gives its {@link Title} and {@link Actions} a
 * shared type scale, so their sizes stay in step.
 *
 * @example
 * <Header.Header level="h4">
 *   <Header.Title>Ranges</Header.Title>
 *   <Header.Actions><Button.Button onClick={add}><Icon.Add /></Button.Button></Header.Actions>
 * </Header.Header>
 * @param props.level - The font level for the header. See the {@link Typography.Text}
 * component for all possible levels. Default is "h1."
 * @param props.divided - If true, creates a divider between the start icon, header
 * text, and each action. Default is false.
 */
export declare const Header: ({ className, level, divided, bordered, ...rest }: HeaderProps) => ReactElement;
//# sourceMappingURL=Header.d.ts.map