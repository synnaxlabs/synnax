import "./Container.css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { type Input } from "@synnaxlabs/lyra/input";
import { sticky } from "@synnaxlabs/x";
import { type ReactElement } from "react";
export interface ContainerProps extends Omit<Flex.BoxProps, "onChange">, Partial<Input.OptionalControl<sticky.XY>> {
    dragEnabled?: boolean;
    initial?: sticky.XY;
}
export declare const Container: import("react").MemoExoticComponent<({ className, value, onChange, style, draggable, initial, ...rest }: ContainerProps) => ReactElement | null>;
//# sourceMappingURL=Container.d.ts.map