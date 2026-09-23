import { Select } from "@synnaxlabs/lyra/select";
import { type optional } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { type PrimitiveTypeName } from "./primitive";
export interface SelectTypeProps extends optional.Optional<Select.ButtonsProps<PrimitiveTypeName>, "keys"> {
}
export declare const SelectType: (props: SelectTypeProps) => ReactElement;
//# sourceMappingURL=SelectType.d.ts.map