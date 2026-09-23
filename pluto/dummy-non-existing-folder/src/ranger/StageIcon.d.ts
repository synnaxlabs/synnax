import { type Icon } from "@synnaxlabs/lyra/icon";
import { type CrudeTimeRange } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface StageIconProps extends Icon.IconProps {
    timeRange: CrudeTimeRange;
}
export declare const StageIcon: ({ timeRange, ...rest }: StageIconProps) => ReactElement;
//# sourceMappingURL=StageIcon.d.ts.map