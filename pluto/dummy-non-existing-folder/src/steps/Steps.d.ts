import "./Steps.css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type Input } from "@synnaxlabs/lyra/input";
import { type ReactElement } from "react";
export interface Step {
    key: string;
    title: string;
}
export interface StepsProps extends Omit<Flex.BoxProps, "children" | "onChange">, Input.Control<string> {
    steps: Step[];
}
export declare const Steps: ({ steps, value, onChange, ...rest }: StepsProps) => ReactElement;
//# sourceMappingURL=Steps.d.ts.map