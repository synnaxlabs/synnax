import { type CrudeTimeSpan, type TimeSpanStringFormat } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Generic } from "../../generic";
import { Text } from "../../text";
export type TimeSpanProps<E extends Generic.ElementType = "p"> = Omit<Text.TextProps<E>, "children"> & {
    children: CrudeTimeSpan;
    format?: TimeSpanStringFormat;
};
export declare const TimeSpan: <E extends Generic.ElementType = "p">({ children, format, ...rest }: TimeSpanProps<E>) => ReactElement;
//# sourceMappingURL=TimeSpan.d.ts.map