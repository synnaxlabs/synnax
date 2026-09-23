import { type ReactElement } from "react";
export interface RadiusFieldsProps {
    path: string;
}
/**
 * RadiusFields edits a symbol's corner radius through one x and one y percentage,
 * writing the pair into every corner. The stored shape carries corners because the
 * renderer clips against them; no symbol has ever varied the radius by corner.
 */
export declare const RadiusFields: ({ path }: RadiusFieldsProps) => ReactElement | null;
//# sourceMappingURL=Radius.d.ts.map