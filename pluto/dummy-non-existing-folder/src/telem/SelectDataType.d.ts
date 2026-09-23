import { Select } from "@synnaxlabs/lyra/select";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface SelectDataTypeProps extends Omit<Select.StaticProps<string>, "data" | "resourceName"> {
    hideVariableDensity?: boolean;
    hideDataTypes?: DataType[];
}
export declare const SelectDataType: ({ hideVariableDensity, hideDataTypes, ...rest }: SelectDataTypeProps) => ReactElement;
//# sourceMappingURL=SelectDataType.d.ts.map