import "./Time.css";
import { type ReactElement, type ReactNode } from "react";
export interface EffectProps {
    /** What a commit would change, or null when it would change nothing else. */
    children: ReactNode;
    className?: string;
}
/**
 * A footer that says what a commit would change. It expands in when `children` turns
 * non-null and collapses out, still showing its last content, when they turn null.
 */
export declare const Effect: ({ children, className }: EffectProps) => ReactElement;
//# sourceMappingURL=Effect.d.ts.map