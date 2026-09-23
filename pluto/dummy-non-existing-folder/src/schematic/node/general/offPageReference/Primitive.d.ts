import "./offPageReference.css";
import { type schematic } from "@synnaxlabs/client";
import { color, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export declare const offPageReferenceTooltip: (page?: schematic.Page, dblClickNavDisabled?: boolean) => string | undefined;
interface RenderProps extends Partial<Pick<schematic.OffPageReferenceNodeConfig, "orientation">> {
    level?: text.Level;
    color?: color.Crude;
    id?: string;
    label?: string;
    className?: string;
    title?: string;
    linked?: boolean;
    pageType?: schematic.PageType;
    onLabelChange?: (label: string) => void;
}
export declare const OffPageReference: ({ id, className, orientation, label, color: colorVal, level, linked, pageType, onLabelChange, }: RenderProps) => ReactElement;
export {};
//# sourceMappingURL=Primitive.d.ts.map