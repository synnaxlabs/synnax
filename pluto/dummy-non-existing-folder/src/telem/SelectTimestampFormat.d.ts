import "./SelectTimestampFormat.css";
import { Select } from "@synnaxlabs/lyra/select";
import { type TimestampFormat } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface SelectTimestampFormatProps extends Omit<Select.StaticProps<TimestampFormat>, "data" | "resourceName"> {
}
export declare const SelectTimestampFormat: ({ dialogProps, ...rest }: SelectTimestampFormatProps) => ReactElement;
//# sourceMappingURL=SelectTimestampFormat.d.ts.map