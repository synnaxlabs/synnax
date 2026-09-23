import { type CrudeTimeStamp, type TimestampFormat, type TimeZone } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type Generic } from "../../generic";
import { Text } from "../../text";
export type TimeStampProps<E extends Generic.ElementType = "p"> = Omit<Text.TextProps<E>, "children"> & {
    children: CrudeTimeStamp;
    format?: TimestampFormat;
    suppliedTimeZone?: TimeZone;
    displayTimeZone?: TimeZone;
};
export declare const TimeStamp: <E extends Generic.ElementType = "p">({ format, suppliedTimeZone, displayTimeZone, children, ...rest }: TimeStampProps<E>) => ReactElement;
//# sourceMappingURL=TimeStamp.d.ts.map