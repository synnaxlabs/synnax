import { type ReactElement } from "react";
import { type BaseProps } from "./Base";
export interface LogProps extends Omit<BaseProps, "channels" | "telem" | "channelNamesHidden" | "receiptTimestampHidden" | "timestampPrecision"> {
}
export declare const Log: ({ enableTriggers, ...rest }: LogProps) => ReactElement | null;
//# sourceMappingURL=Log.d.ts.map