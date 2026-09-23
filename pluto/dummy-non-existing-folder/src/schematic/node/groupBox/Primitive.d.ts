import "./groupBox.css";
import { type ReactElement } from "react";
export interface PrimitiveProps {
    nodeKey: string;
    members: string[];
    className?: string;
}
/**
 * Primitive renders the group's box, sized from the members' rendered bounds
 * rather than stored state.
 */
export declare const Primitive: ({ nodeKey, members, className, }: PrimitiveProps) => ReactElement;
//# sourceMappingURL=Primitive.d.ts.map