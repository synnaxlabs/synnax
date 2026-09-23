import { type ReactElement, type ReactNode } from "react";
export interface InProps {
    /**
     * itemKey identifies the content so an {@link Out} with the same key can
     * host it. Must be unique within the enclosing Context.
     */
    itemKey: string;
    /** children renders the content registered under itemKey. */
    children: ReactNode;
}
/**
 * In renders children into a detached element registered under itemKey in the enclosing
 * {@link Context}. The content stays mounted at the In's position in the React tree for
 * the In's whole lifetime, while {@link Out} parts with the same key host it in the
 * DOM. Because the element is moved rather than recreated when its host changes, the
 * content keeps its state (DOM, WebGL contexts) across moves.
 */
export declare const In: ({ itemKey, children }: InProps) => ReactElement;
//# sourceMappingURL=In.d.ts.map