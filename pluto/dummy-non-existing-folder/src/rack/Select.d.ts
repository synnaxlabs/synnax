import { type rack } from "@synnaxlabs/client";
import { type Dialog } from "@synnaxlabs/lyra/dialog";
import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";
import { type Flux } from "../flux";
import { type ListQuery } from "./queries";
export interface SelectSingleProps extends Omit<Select.SingleFrameProps<rack.Key, rack.Payload | undefined>, "data">, Flux.UseListParams<ListQuery, rack.Key, rack.Payload>, Omit<Dialog.FrameProps, "onChange">, Pick<Select.DialogProps<rack.Key>, "emptyContent">, Pick<Select.SingleProps<rack.Key, rack.Payload | undefined>, "preview" | "triggerProps"> {
}
export declare const SelectSingle: ({ filter, initialQuery, ...rest }: SelectSingleProps) => ReactElement;
//# sourceMappingURL=Select.d.ts.map