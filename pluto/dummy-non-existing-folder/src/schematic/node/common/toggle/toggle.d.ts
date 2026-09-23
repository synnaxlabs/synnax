import { schematic } from "@synnaxlabs/client";
import { type FC } from "react";
import { Grid } from "../grid";
import { type ButtonProps } from "./Button";
import { type NodeProps } from "../../spec";
export declare const toggleConfigZ: import("zod").ZodObject<{
    stalenessTimeout: import("zod").ZodDefault<import("zod").ZodNumber>;
    stalenessColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>, import("zod").ZodObject<{
        rgba255: import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>], null>;
    }, import("zod/v4/core").$strip>, import("zod").ZodObject<{
        r: import("zod").ZodInt;
        g: import("zod").ZodInt;
        b: import("zod").ZodInt;
        a: import("zod").ZodPipe<import("zod").ZodNumber, import("zod").ZodTransform<number, number>>;
    }, import("zod/v4/core").$strip>]>, import("zod").ZodTransform<[number, number, number, number], string | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | {
        rgba255: [number, number, number, number];
    } | {
        r: number;
        g: number;
        b: number;
        a: number;
    } | [number, number, number] | [number, number, number, number]>>>;
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
    stateChannel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    commandChannel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    control: import("zod").ZodOptional<import("zod").ZodObject<{
        authority: import("zod").ZodOptional<import("zod").ZodInt>;
        hidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        chipHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        indicatorHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
            bottom: "bottom";
            center: "center";
            left: "left";
            right: "right";
            top: "top";
        }>>;
    }, import("zod/v4/core").$strip>>;
    onClickDelay: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>;
export type ToggleConfig = schematic.ToggleConfig;
export declare const createToggle: <C extends ToggleConfig>(BaseSymbol: FC<Omit<C, "label"> & ButtonProps>, overrides?: {
    grid?: Partial<Omit<Grid.GridProps, "editable">>;
}) => FC<NodeProps<C>>;
export type DummyToggleConfig = Omit<schematic.DummyToggleSymbolConfig, "color">;
export declare const createDummyToggle: <C extends DummyToggleConfig>(Primitive: FC<Omit<C, "label"> & ButtonProps>) => FC<NodeProps<C>>;
//# sourceMappingURL=toggle.d.ts.map