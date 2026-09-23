import { type CrudeTimeRange, type TimeZone } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { Flex } from "../../flex";
import { Text } from "../../text";
export interface TimeRangeProps extends Omit<Flex.BoxProps<"div">, "children">, Pick<Text.TextProps, "level" | "color" | "weight"> {
    children: CrudeTimeRange;
    displayTimeZone?: TimeZone;
}
export declare const TimeRange: ({ children, level, color, displayTimeZone, weight, ...rest }: TimeRangeProps) => ReactElement | null;
//# sourceMappingURL=TimeRange.d.ts.map