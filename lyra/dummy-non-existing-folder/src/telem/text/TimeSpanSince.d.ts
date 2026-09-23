import { type CrudeTimeStamp, type TimeSpan as XTimeSpan, type TimeSpanStringFormat } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Generic } from "../../generic";
import { type TimeSpanProps } from "./TimeSpan";
export type TimeSpanSinceProps<E extends Generic.ElementType = "p"> = Omit<TimeSpanProps<E>, "children"> & {
    children: CrudeTimeStamp;
    format?: TimeSpanStringFormat;
};
export declare const useTimeSpanSince: (stamp: CrudeTimeStamp) => XTimeSpan;
export declare const TimeSpanSince: <E extends Generic.ElementType = "p">({ children, ...rest }: TimeSpanSinceProps<E>) => ReactElement;
//# sourceMappingURL=TimeSpanSince.d.ts.map