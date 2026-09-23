import { type ReactElement } from "react";
export interface FieldsProps {
    /** Path to the config holding the staleness keys. Defaults to the form root. */
    path?: string;
}
/**
 * Fields edits the color a component takes on, and the delay before it does, once its
 * source stops sending. It renders as a pair of siblings, so the caller places it in a
 * row of its own choosing. An unchosen color is absent, so the swatch shows the theme
 * color it resolves to until a pick writes one.
 */
export declare const Fields: ({ path }?: FieldsProps) => ReactElement;
//# sourceMappingURL=Fields.d.ts.map