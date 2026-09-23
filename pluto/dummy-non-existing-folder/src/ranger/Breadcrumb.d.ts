import { type ranger } from "@synnaxlabs/client";
import { Breadcrumb as Base } from "@synnaxlabs/lyra/breadcrumb";
import { type CrudeTimeRange } from "@synnaxlabs/x";
export interface BreadcrumbProps extends Omit<Base.BreadcrumbProps, "children"> {
    timeRange?: CrudeTimeRange;
    name: string;
    showParent?: boolean;
    parent?: Pick<ranger.Payload, "name"> | null;
    /** DOM id for the name, so {@link Text.edit} can target it. */
    nameID?: string;
    /** Makes the name editable in place. Absent, the name is plain text. */
    onRename?: (name: string) => void;
}
export declare const Breadcrumb: ({ timeRange, name, parent, showParent, nameID, onRename, ...rest }: BreadcrumbProps) => import("react").JSX.Element;
//# sourceMappingURL=Breadcrumb.d.ts.map