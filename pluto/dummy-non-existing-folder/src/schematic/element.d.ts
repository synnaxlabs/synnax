import { schematic } from "@synnaxlabs/client";
import { Edge } from "./edge";
import { Node } from "./node";
export declare const elementConfigZ: import("zod").ZodDiscriminatedUnion<[import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"cap">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"filter">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flow_straightener">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"heater_element">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"iso_cap">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"iso_filter">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"nozzle">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"orifice">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"orifice_plate">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"strainer">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"strainer_cone">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"thruster">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"vent">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_general">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_electromagnetic">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_variable_area">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_coriolis">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_nozzle">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_venturi">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_ring_piston">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_positive_displacement">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_turbine">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_pulse">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_float_sensor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flowmeter_orifice">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"box">;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    backgroundColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    dimensions: import("zod").ZodPrefault<import("zod").ZodObject<{
        width: import("zod").ZodNumber;
        height: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    borderRadius: import("zod").ZodDefault<import("zod").ZodNumber>;
    strokeWidth: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"button">;
    size: import("zod").ZodDefault<import("zod").ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    level: import("zod").ZodOptional<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    onClickDelay: import("zod").ZodDefault<import("zod").ZodNumber>;
    commandChannel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    mode: import("zod").ZodDefault<import("zod").ZodEnum<{
        fire: "fire";
        momentary: "momentary";
        pulse: "pulse";
    }>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"circle">;
    radius: import("zod").ZodDefault<import("zod").ZodNumber>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    backgroundColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    strokeWidth: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    rollingAverage: import("zod").ZodOptional<import("zod").ZodInt32>;
    precision: import("zod").ZodDefault<import("zod").ZodNumber>;
    notation: import("zod").ZodDefault<import("zod").ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
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
    variant: import("zod").ZodLiteral<"gauge">;
    position: import("zod").ZodOptional<import("zod").ZodObject<{
        x: import("zod").ZodNumber;
        y: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    bounds: import("zod").ZodPrefault<import("zod").ZodObject<{
        lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
    }, import("zod/v4/core").$strip>>;
    barWidth: import("zod").ZodDefault<import("zod").ZodNumber>;
    location: import("zod").ZodPrefault<import("zod").ZodObject<{
        x: import("zod").ZodEnum<{
            center: "center";
            left: "left";
            right: "right";
        }>;
        y: import("zod").ZodEnum<{
            bottom: "bottom";
            center: "center";
            top: "top";
        }>;
    }, import("zod/v4/core").$strip>>;
    units: import("zod").ZodDefault<import("zod").ZodString>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"input">;
    size: import("zod").ZodDefault<import("zod").ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    commandChannel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    dimensions: import("zod").ZodOptional<import("zod").ZodObject<{
        width: import("zod").ZodNumber;
        height: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    disabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    onClickDelay: import("zod").ZodDefault<import("zod").ZodNumber>;
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
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"light">;
    channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    threshold: import("zod").ZodOptional<import("zod").ZodObject<{
        lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    variant: import("zod").ZodLiteral<"line">;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    start: import("zod").ZodPrefault<import("zod").ZodObject<{
        x: import("zod").ZodNumber;
        y: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    end: import("zod").ZodPrefault<import("zod").ZodObject<{
        x: import("zod").ZodNumber;
        y: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    strokeWidth: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"off_page_reference">;
    orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    page: import("zod").ZodOptional<import("zod").ZodObject<{
        type: import("zod").ZodEnum<{
            lineplot: "lineplot";
            log: "log";
            schematic: "schematic";
            table: "table";
        }>;
        key: import("zod").ZodString;
    }, import("zod/v4/core").$strip>>;
    dblClickNavDisabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"polygon">;
    numSides: import("zod").ZodDefault<import("zod").ZodNumber>;
    sideLength: import("zod").ZodDefault<import("zod").ZodNumber>;
    rotation: import("zod").ZodDefault<import("zod").ZodNumber>;
    cornerRounding: import("zod").ZodDefault<import("zod").ZodNumber>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    backgroundColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    strokeWidth: import("zod").ZodDefault<import("zod").ZodNumber>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"select">;
    size: import("zod").ZodDefault<import("zod").ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    commandChannel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    inlineSize: import("zod").ZodDefault<import("zod").ZodNumber>;
    options: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        key: import("zod").ZodString;
        name: import("zod").ZodString;
        value: import("zod").ZodNumber;
        color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    }, import("zod/v4/core").$strip>>>;
    disabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    onClickDelay: import("zod").ZodDefault<import("zod").ZodNumber>;
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
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    scale: import("zod").ZodDefault<import("zod").ZodNumber>;
    variant: import("zod").ZodLiteral<"scale">;
    orientation: import("zod").ZodDefault<import("zod").ZodEnum<{
        bottom: "bottom";
        left: "left";
        right: "right";
        top: "top";
    }>>;
    position: import("zod").ZodOptional<import("zod").ZodObject<{
        x: import("zod").ZodNumber;
        y: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    dimensions: import("zod").ZodPrefault<import("zod").ZodObject<{
        width: import("zod").ZodNumber;
        height: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    indicator: import("zod").ZodPrefault<import("zod").ZodObject<{
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
        channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
        rollingAverage: import("zod").ZodOptional<import("zod").ZodInt32>;
        precision: import("zod").ZodDefault<import("zod").ZodNumber>;
        notation: import("zod").ZodDefault<import("zod").ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        bounds: import("zod").ZodPrefault<import("zod").ZodObject<{
            lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
            upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        }, import("zod/v4/core").$strip>>;
        color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        axisColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        textColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        units: import("zod").ZodDefault<import("zod").ZodString>;
        fillHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        caretHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        scaleHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        side: import("zod").ZodDefault<import("zod").ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        caretSide: import("zod").ZodDefault<import("zod").ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        level: import("zod").ZodDefault<import("zod").ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
    }, import("zod/v4/core").$strip>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"setpoint">;
    size: import("zod").ZodDefault<import("zod").ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
    commandChannel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    dimensions: import("zod").ZodOptional<import("zod").ZodObject<{
        width: import("zod").ZodNumber;
        height: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    units: import("zod").ZodDefault<import("zod").ZodString>;
    disabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    onClickDelay: import("zod").ZodDefault<import("zod").ZodNumber>;
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
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"state_indicator">;
    channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    inlineSize: import("zod").ZodDefault<import("zod").ZodNumber>;
    options: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        key: import("zod").ZodString;
        name: import("zod").ZodString;
        value: import("zod").ZodNumber;
        color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    }, import("zod/v4/core").$strip>>>;
    size: import("zod").ZodDefault<import("zod").ZodEnum<{
        huge: "huge";
        large: "large";
        medium: "medium";
        small: "small";
        tiny: "tiny";
    }>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"string_display">;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    textColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    tooltip: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodString>>;
    inlineSize: import("zod").ZodDefault<import("zod").ZodNumber>;
    channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"switch">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"text_box">;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    width: import("zod").ZodDefault<import("zod").ZodNumber>;
    align: import("zod").ZodDefault<import("zod").ZodEnum<{
        center: "center";
        end: "end";
        start: "start";
        stretch: "stretch";
    }>>;
    autoFitDisabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    value: import("zod").ZodDefault<import("zod").ZodString>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
    rollingAverage: import("zod").ZodOptional<import("zod").ZodInt32>;
    precision: import("zod").ZodDefault<import("zod").ZodNumber>;
    notation: import("zod").ZodDefault<import("zod").ZodEnum<{
        engineering: "engineering";
        scientific: "scientific";
        standard: "standard";
    }>>;
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
    variant: import("zod").ZodLiteral<"value">;
    position: import("zod").ZodOptional<import("zod").ZodObject<{
        x: import("zod").ZodNumber;
        y: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    textColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    tooltip: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodString>>;
    redline: import("zod").ZodPrefault<import("zod").ZodObject<{
        bounds: import("zod").ZodObject<{
            lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
            upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        }, import("zod/v4/core").$strip>;
        gradient: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
            key: import("zod").ZodString;
            color: import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
            }, import("zod/v4/core").$strip>]>;
            position: import("zod").ZodNumber;
            switched: import("zod").ZodOptional<import("zod").ZodBoolean>;
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>;
    units: import("zod").ZodDefault<import("zod").ZodString>;
    inlineSize: import("zod").ZodDefault<import("zod").ZodNumber>;
    level: import("zod").ZodDefault<import("zod").ZodEnum<{
        h1: "h1";
        h2: "h2";
        h3: "h3";
        h4: "h4";
        h5: "h5";
        p: "p";
        small: "small";
    }>>;
    location: import("zod").ZodPrefault<import("zod").ZodObject<{
        x: import("zod").ZodEnum<{
            center: "center";
            left: "left";
            right: "right";
        }>;
        y: import("zod").ZodEnum<{
            bottom: "bottom";
            center: "center";
            top: "top";
        }>;
    }, import("zod/v4/core").$strip>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"agitator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"cross_beam_agitator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flat_blade_agitator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"heat_exchanger_general">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"heat_exchanger_m">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"heat_exchanger_straight_tube">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"helical_agitator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"paddle_agitator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"propeller_agitator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"rotary_mixer">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"static_mixer">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"cavity_pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"centrifugal_compressor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"compressor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"diaphragm_pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"ejection_pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"ejector_compressor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"liquid_ring_compressor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"piston_pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"roller_vane_compressor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"screw_pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"turbo_compressor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"vacuum_pump">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"burst_disc">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flame_arrestor">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flame_arrestor_detonation">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flame_arrestor_explosion">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flame_arrestor_fire_res">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"flame_arrestor_fire_res_detonation">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"iso_burst_disc">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"angled_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"angled_relief_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"angled_spring_loaded_relief_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"ball_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"breather_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"butterfly_valve_one">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"butterfly_valve_two">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"check_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"check_valve_with_arrow">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"electric_regulator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"electric_regulator_motorized">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"four_way_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"gate_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"iso_check_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"manual_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"needle_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"regulator">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"regulator_manual">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"relief_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"solenoid_valve">;
    normallyOpen: import("zod").ZodDefault<import("zod").ZodBoolean>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    enabled: import("zod").ZodDefault<import("zod").ZodBoolean>;
    clickable: import("zod").ZodDefault<import("zod").ZodBoolean>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"spring_loaded_relief_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"three_way_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"three_way_ball_valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"valve">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"cross_junction">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"cylinder">;
    dimensions: import("zod").ZodPrefault<import("zod").ZodObject<{
        width: import("zod").ZodNumber;
        height: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    borderRadius: import("zod").ZodOptional<import("zod").ZodObject<{
        topLeft: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
        topRight: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
        bottomLeft: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
        bottomRight: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    backgroundColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"tank">;
    position: import("zod").ZodOptional<import("zod").ZodObject<{
        x: import("zod").ZodNumber;
        y: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    backgroundColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    dimensions: import("zod").ZodPrefault<import("zod").ZodObject<{
        width: import("zod").ZodNumber;
        height: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>;
    borderRadius: import("zod").ZodPrefault<import("zod").ZodObject<{
        topLeft: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
        topRight: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
        bottomLeft: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
        bottomRight: import("zod").ZodObject<{
            x: import("zod").ZodNumber;
            y: import("zod").ZodNumber;
        }, import("zod/v4/core").$strip>;
    }, import("zod/v4/core").$strip>>;
    fill: import("zod").ZodPrefault<import("zod").ZodObject<{
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
        channel: import("zod").ZodOptional<import("zod").ZodUnion<[import("zod").ZodUInt32, import("zod").ZodPipe<import("zod").ZodString, import("zod").ZodTransform<number, string>>]>>;
        rollingAverage: import("zod").ZodOptional<import("zod").ZodInt32>;
        precision: import("zod").ZodDefault<import("zod").ZodNumber>;
        notation: import("zod").ZodDefault<import("zod").ZodEnum<{
            engineering: "engineering";
            scientific: "scientific";
            standard: "standard";
        }>>;
        bounds: import("zod").ZodPrefault<import("zod").ZodObject<{
            lower: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
            upper: import("zod").ZodNumber | import("zod").ZodType<number, unknown, import("zod/v4/core").$ZodTypeInternals<number, unknown>>;
        }, import("zod/v4/core").$strip>>;
        color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        axisColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        textColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        units: import("zod").ZodDefault<import("zod").ZodString>;
        fillHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        caretHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        scaleHidden: import("zod").ZodDefault<import("zod").ZodBoolean>;
        side: import("zod").ZodDefault<import("zod").ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        caretSide: import("zod").ZodDefault<import("zod").ZodEnum<{
            bottom: "bottom";
            left: "left";
            right: "right";
            top: "top";
        }>>;
        level: import("zod").ZodDefault<import("zod").ZodEnum<{
            h1: "h1";
            h2: "h2";
            h3: "h3";
            h4: "h4";
            h5: "h5";
            p: "p";
            small: "small";
        }>>;
    }, import("zod/v4/core").$strip>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"t_junction">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"custom_actuator">;
    specKey: import("zod").ZodString;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    stateOverrides: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        key: import("zod").ZodString;
        name: import("zod").ZodString;
        regions: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
            key: import("zod").ZodString;
            name: import("zod").ZodString;
            selectors: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodString>>;
            strokeColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
            fillColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
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
    variant: import("zod").ZodLiteral<"custom_static">;
    specKey: import("zod").ZodString;
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    stateOverrides: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        key: import("zod").ZodString;
        name: import("zod").ZodString;
        regions: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
            key: import("zod").ZodString;
            name: import("zod").ZodString;
            selectors: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodString>>;
            strokeColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
            fillColor: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
        }, import("zod/v4/core").$strip>>>;
    }, import("zod/v4/core").$strip>>>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    variant: import("zod").ZodLiteral<"group_box">;
    members: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodString>>;
    locked: import("zod").ZodDefault<import("zod").ZodBoolean>;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"pipe">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"electric">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"secondary">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"jacketed">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"hydraulic">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"pneumatic">;
}, import("zod/v4/core").$strip>, import("zod").ZodObject<{
    color: import("zod").ZodOptional<import("zod").ZodPipe<import("zod").ZodUnion<readonly [import("zod").ZodString, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt], null>, import("zod").ZodTuple<[import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodInt, import("zod").ZodNumber], null>, import("zod").ZodTuple<[import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber, import("zod").ZodNumber], null>, import("zod").ZodObject<{
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
    segments: import("zod").ZodDefault<import("zod").ZodArray<import("zod").ZodObject<{
        direction: import("zod").ZodEnum<{
            x: "x";
            y: "y";
        }>;
        length: import("zod").ZodNumber;
    }, import("zod/v4/core").$strip>>>;
    variant: import("zod").ZodLiteral<"data">;
}, import("zod/v4/core").$strip>], "variant">;
export type ElementConfig = schematic.ElementConfig;
export declare const ELEMENT_REGISTRY: {
    cap: Node.Spec<"cap", import("./node/common/create").StaticConfig<"cap">>;
    filter: Node.Spec<"filter", import("./node/common/create").StaticConfig<"filter">>;
    flow_straightener: Node.Spec<"flow_straightener", import("./node/common/create").StaticConfig<"flow_straightener">>;
    heater_element: Node.Spec<"heater_element", import("./node/common/create").StaticConfig<"heater_element">>;
    iso_cap: Node.Spec<"iso_cap", import("./node/common/create").StaticConfig<"iso_cap">>;
    iso_filter: Node.Spec<"iso_filter", import("./node/common/create").StaticConfig<"iso_filter">>;
    nozzle: Node.Spec<"nozzle", import("./node/common/create").StaticConfig<"nozzle">>;
    orifice: Node.Spec<"orifice", import("./node/common/create").StaticConfig<"orifice">>;
    orifice_plate: Node.Spec<"orifice_plate", import("./node/common/create").StaticConfig<"orifice_plate">>;
    strainer: Node.Spec<"strainer", import("./node/common/create").StaticConfig<"strainer">>;
    strainer_cone: Node.Spec<"strainer_cone", import("./node/common/create").StaticConfig<"strainer_cone">>;
    thruster: Node.Spec<"thruster", import("./node/common/create").ToggleSymbolConfig<"thruster">>;
    vent: Node.Spec<"vent", import("./node/common/create").StaticConfig<"vent">>;
    flowmeter_general: Node.Spec<"flowmeter_general", import("./node/common/create").StaticConfig<"flowmeter_general">>;
    flowmeter_electromagnetic: Node.Spec<"flowmeter_electromagnetic", import("./node/common/create").StaticConfig<"flowmeter_electromagnetic">>;
    flowmeter_variable_area: Node.Spec<"flowmeter_variable_area", import("./node/common/create").StaticConfig<"flowmeter_variable_area">>;
    flowmeter_coriolis: Node.Spec<"flowmeter_coriolis", import("./node/common/create").StaticConfig<"flowmeter_coriolis">>;
    flowmeter_nozzle: Node.Spec<"flowmeter_nozzle", import("./node/common/create").StaticConfig<"flowmeter_nozzle">>;
    flowmeter_venturi: Node.Spec<"flowmeter_venturi", import("./node/common/create").StaticConfig<"flowmeter_venturi">>;
    flowmeter_ring_piston: Node.Spec<"flowmeter_ring_piston", import("./node/common/create").StaticConfig<"flowmeter_ring_piston">>;
    flowmeter_positive_displacement: Node.Spec<"flowmeter_positive_displacement", import("./node/common/create").StaticConfig<"flowmeter_positive_displacement">>;
    flowmeter_turbine: Node.Spec<"flowmeter_turbine", import("./node/common/create").StaticConfig<"flowmeter_turbine">>;
    flowmeter_pulse: Node.Spec<"flowmeter_pulse", import("./node/common/create").StaticConfig<"flowmeter_pulse">>;
    flowmeter_float_sensor: Node.Spec<"flowmeter_float_sensor", import("./node/common/create").StaticConfig<"flowmeter_float_sensor">>;
    flowmeter_orifice: Node.Spec<"flowmeter_orifice", import("./node/common/create").StaticConfig<"flowmeter_orifice">>;
    box: Node.Spec<"box", schematic.BoxNodeConfig>;
    button: Node.Spec<"button", schematic.ButtonNodeConfig>;
    circle: Node.Spec<"circle", schematic.CircleNodeConfig>;
    gauge: Node.Spec<"gauge", schematic.GaugeNodeConfig>;
    input: Node.Spec<"input", schematic.InputNodeConfig>;
    light: Node.Spec<"light", schematic.LightNodeConfig>;
    line: Node.Spec<"line", schematic.LineNodeConfig>;
    off_page_reference: Node.Spec<"off_page_reference", schematic.OffPageReferenceNodeConfig>;
    polygon: Node.Spec<"polygon", schematic.PolygonNodeConfig>;
    scale: Node.Spec<"scale", schematic.ScaleNodeConfig>;
    select: Node.Spec<"select", schematic.SelectNodeConfig>;
    setpoint: Node.Spec<"setpoint", schematic.SetpointNodeConfig>;
    state_indicator: Node.Spec<"state_indicator", schematic.StateIndicatorNodeConfig>;
    string_display: Node.Spec<"string_display", schematic.StringDisplayNodeConfig>;
    switch: Node.Spec<"switch", schematic.SwitchNodeConfig>;
    text_box: Node.Spec<"text_box", schematic.TextBoxNodeConfig>;
    value: Node.Spec<"value", schematic.ValueNodeConfig>;
    agitator: Node.Spec<"agitator", import("./node/common/create").ToggleSymbolConfig<"agitator">>;
    cross_beam_agitator: Node.Spec<"cross_beam_agitator", import("./node/common/create").ToggleSymbolConfig<"cross_beam_agitator">>;
    flat_blade_agitator: Node.Spec<"flat_blade_agitator", import("./node/common/create").ToggleSymbolConfig<"flat_blade_agitator">>;
    heat_exchanger_general: Node.Spec<"heat_exchanger_general", import("./node/common/create").StaticConfig<"heat_exchanger_general">>;
    heat_exchanger_m: Node.Spec<"heat_exchanger_m", import("./node/common/create").StaticConfig<"heat_exchanger_m">>;
    heat_exchanger_straight_tube: Node.Spec<"heat_exchanger_straight_tube", import("./node/common/create").StaticConfig<"heat_exchanger_straight_tube">>;
    helical_agitator: Node.Spec<"helical_agitator", import("./node/common/create").ToggleSymbolConfig<"helical_agitator">>;
    paddle_agitator: Node.Spec<"paddle_agitator", import("./node/common/create").ToggleSymbolConfig<"paddle_agitator">>;
    propeller_agitator: Node.Spec<"propeller_agitator", import("./node/common/create").ToggleSymbolConfig<"propeller_agitator">>;
    rotary_mixer: Node.Spec<"rotary_mixer", import("./node/common/create").ToggleSymbolConfig<"rotary_mixer">>;
    static_mixer: Node.Spec<"static_mixer", import("./node/common/create").StaticConfig<"static_mixer">>;
    cavity_pump: Node.Spec<"cavity_pump", import("./node/common/create").ToggleSymbolConfig<"cavity_pump">>;
    centrifugal_compressor: Node.Spec<"centrifugal_compressor", import("./node/common/create").ToggleSymbolConfig<"centrifugal_compressor">>;
    compressor: Node.Spec<"compressor", import("./node/common/create").ToggleSymbolConfig<"compressor">>;
    diaphragm_pump: Node.Spec<"diaphragm_pump", import("./node/common/create").ToggleSymbolConfig<"diaphragm_pump">>;
    ejection_pump: Node.Spec<"ejection_pump", import("./node/common/create").ToggleSymbolConfig<"ejection_pump">>;
    ejector_compressor: Node.Spec<"ejector_compressor", import("./node/common/create").ToggleSymbolConfig<"ejector_compressor">>;
    liquid_ring_compressor: Node.Spec<"liquid_ring_compressor", import("./node/common/create").ToggleSymbolConfig<"liquid_ring_compressor">>;
    piston_pump: Node.Spec<"piston_pump", import("./node/common/create").ToggleSymbolConfig<"piston_pump">>;
    pump: Node.Spec<"pump", import("./node/common/create").ToggleSymbolConfig<"pump">>;
    roller_vane_compressor: Node.Spec<"roller_vane_compressor", import("./node/common/create").ToggleSymbolConfig<"roller_vane_compressor">>;
    screw_pump: Node.Spec<"screw_pump", import("./node/common/create").ToggleSymbolConfig<"screw_pump">>;
    turbo_compressor: Node.Spec<"turbo_compressor", import("./node/common/create").ToggleSymbolConfig<"turbo_compressor">>;
    vacuum_pump: Node.Spec<"vacuum_pump", import("./node/common/create").ToggleSymbolConfig<"vacuum_pump">>;
    burst_disc: Node.Spec<"burst_disc", import("./node/common/create").StaticConfig<"burst_disc">>;
    flame_arrestor: Node.Spec<"flame_arrestor", import("./node/common/create").StaticConfig<"flame_arrestor">>;
    flame_arrestor_detonation: Node.Spec<"flame_arrestor_detonation", import("./node/common/create").StaticConfig<"flame_arrestor_detonation">>;
    flame_arrestor_explosion: Node.Spec<"flame_arrestor_explosion", import("./node/common/create").StaticConfig<"flame_arrestor_explosion">>;
    flame_arrestor_fire_res: Node.Spec<"flame_arrestor_fire_res", import("./node/common/create").StaticConfig<"flame_arrestor_fire_res">>;
    flame_arrestor_fire_res_detonation: Node.Spec<"flame_arrestor_fire_res_detonation", import("./node/common/create").StaticConfig<"flame_arrestor_fire_res_detonation">>;
    iso_burst_disc: Node.Spec<"iso_burst_disc", import("./node/common/create").StaticConfig<"iso_burst_disc">>;
    angled_valve: Node.Spec<"angled_valve", import("./node/common/create").ToggleSymbolConfig<"angled_valve">>;
    angled_relief_valve: Node.Spec<"angled_relief_valve", import("./node/common/create").DummyToggleConfig<"angled_relief_valve">>;
    angled_spring_loaded_relief_valve: Node.Spec<"angled_spring_loaded_relief_valve", import("./node/common/create").DummyToggleConfig<"angled_spring_loaded_relief_valve">>;
    ball_valve: Node.Spec<"ball_valve", import("./node/common/create").ToggleSymbolConfig<"ball_valve">>;
    breather_valve: Node.Spec<"breather_valve", import("./node/common/create").DummyToggleConfig<"breather_valve">>;
    butterfly_valve_one: Node.Spec<"butterfly_valve_one", import("./node/common/create").ToggleSymbolConfig<"butterfly_valve_one">>;
    butterfly_valve_two: Node.Spec<"butterfly_valve_two", import("./node/common/create").ToggleSymbolConfig<"butterfly_valve_two">>;
    check_valve: Node.Spec<"check_valve", import("./node/common/create").StaticConfig<"check_valve">>;
    check_valve_with_arrow: Node.Spec<"check_valve_with_arrow", import("./node/common/create").StaticConfig<"check_valve_with_arrow">>;
    electric_regulator: Node.Spec<"electric_regulator", import("./node/common/create").StaticConfig<"electric_regulator">>;
    electric_regulator_motorized: Node.Spec<"electric_regulator_motorized", import("./node/common/create").StaticConfig<"electric_regulator_motorized">>;
    four_way_valve: Node.Spec<"four_way_valve", import("./node/common/create").ToggleSymbolConfig<"four_way_valve">>;
    gate_valve: Node.Spec<"gate_valve", import("./node/common/create").ToggleSymbolConfig<"gate_valve">>;
    iso_check_valve: Node.Spec<"iso_check_valve", import("./node/common/create").StaticConfig<"iso_check_valve">>;
    manual_valve: Node.Spec<"manual_valve", import("./node/common/create").DummyToggleConfig<"manual_valve">>;
    needle_valve: Node.Spec<"needle_valve", import("./node/common/create").DummyToggleConfig<"needle_valve">>;
    regulator: Node.Spec<"regulator", import("./node/common/create").StaticConfig<"regulator">>;
    regulator_manual: Node.Spec<"regulator_manual", import("./node/common/create").StaticConfig<"regulator_manual">>;
    relief_valve: Node.Spec<"relief_valve", import("./node/common/create").DummyToggleConfig<"relief_valve">>;
    solenoid_valve: Node.Spec<"solenoid_valve", schematic.SolenoidValveNodeConfig>;
    spring_loaded_relief_valve: Node.Spec<"spring_loaded_relief_valve", import("./node/common/create").DummyToggleConfig<"spring_loaded_relief_valve">>;
    three_way_valve: Node.Spec<"three_way_valve", import("./node/common/create").ToggleSymbolConfig<"three_way_valve">>;
    three_way_ball_valve: Node.Spec<"three_way_ball_valve", import("./node/common/create").ToggleSymbolConfig<"three_way_ball_valve">>;
    valve: Node.Spec<"valve", import("./node/common/create").ToggleSymbolConfig<"valve">>;
    cross_junction: Node.Spec<"cross_junction", schematic.CrossJunctionNodeConfig>;
    cylinder: Node.Spec<"cylinder", schematic.CylinderNodeConfig>;
    tank: Node.Spec<"tank", schematic.TankNodeConfig>;
    t_junction: Node.Spec<"t_junction", schematic.TJunctionNodeConfig>;
    custom_actuator: Node.Spec<"custom_actuator", import("./node/custom/configs").CustomActuatorConfig>;
    custom_static: Node.Spec<"custom_static", import("./node/custom/configs").CustomStaticConfig>;
    group_box: Node.Spec<"group_box", schematic.GroupBoxNodeConfig>;
    pipe: Edge.Spec<"pipe", schematic.PipeEdgeConfig>;
    electric: Edge.Spec<"electric", schematic.ElectricEdgeConfig>;
    secondary: Edge.Spec<"secondary", schematic.SecondaryEdgeConfig>;
    jacketed: Edge.Spec<"jacketed", schematic.JacketedEdgeConfig>;
    hydraulic: Edge.Spec<"hydraulic", schematic.HydraulicEdgeConfig>;
    pneumatic: Edge.Spec<"pneumatic", schematic.PneumaticEdgeConfig>;
    data: Edge.Spec<"data", schematic.DataEdgeConfig>;
};
//# sourceMappingURL=element.d.ts.map