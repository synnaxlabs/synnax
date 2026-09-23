import { schematic } from "@synnaxlabs/client";
import { type dimensions } from "@synnaxlabs/x";
import { type FC } from "react";
import { Grid } from "../grid";
import { type Primitive } from "../primitive";
import { type NodeProps } from "../../spec";
export declare const configZ: import("zod").ZodObject<{
    label: import("zod").ZodDefault<import("zod").ZodString>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
        bottom: "bottom";
        center: "center";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    direction: import("zod").ZodDefault<import("zod").ZodEnum<{
        x: "x";
        y: "y";
    }>>;
    maxInlineSize: import("zod").ZodDefault<import("zod").ZodNumber>;
    align: import("zod").ZodDefault<import("zod").ZodEnum<{
        center: "center";
        end: "end";
        start: "start";
        stretch: "stretch";
    }>>;
}, import("zod/v4/core").$strip>;
export type Config = schematic.LabelConfig;
export interface LabelProps {
    config: Config;
    onChange?: (next: {
        label: Config;
    }) => void;
}
export declare const Label: FC<Partial<LabelProps>>;
export declare const labeledConfigZ: import("zod").ZodObject<{
    label: import("zod").ZodPrefault<import("zod").ZodObject<{
        label: import("zod").ZodDefault<import("zod").ZodString>;
        level: import("zod").ZodDefault<import("zod").ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
        orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        direction: import("zod").ZodDefault<import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>>;
        maxInlineSize: import("zod").ZodDefault<import("zod").ZodNumber>;
        align: import("zod").ZodDefault<import("zod").ZodEnum<{
            center: "center";
            end: "end";
            start: "start";
            stretch: "stretch";
        }>>;
    }, import("zod/v4/core").$strip>>;
    orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    scale: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>;
export type LabeledConfig = schematic.LabeledConfig;
interface LabeledOverrides<C extends LabeledConfig> {
    grid?: Pick<Grid.GridProps, "allowRotate" | "keepAspectRatio">;
    onResize?: (dimensions: dimensions.Dimensions) => Partial<C>;
}
export declare const createLabeled: <C extends LabeledConfig>(BaseSymbol: FC<Omit<C, "label"> & Primitive.SVGBasedProps>, overrides?: LabeledOverrides<C>) => FC<NodeProps<C>>;
export {};
//# sourceMappingURL=Label.d.ts.map