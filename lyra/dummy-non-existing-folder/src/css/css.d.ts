import { color, direction, location, type spatial, type text } from "@synnaxlabs/x";
import { type ClassValue } from "clsx";
import { type CSSProperties } from "react";
import { CSSGridBuilder } from "./grid";
import { applyCSSVars, removeCSSVars } from "./vars";
import { type Theming } from "../theming";
export declare const 
/** @returns the class name for a block. */
B: (...blocks: string[]) => string, 
/** @returns the class name for an element of the enclosing block. */
E: (element: string) => string, 
/** @returns the class name for a modifier. */
M: (...modifiers: string[]) => string, 
/** @returns the class name for an element of the given block. */
BE: (block: string, ...elements: string[]) => string, 
/** @returns the class name for a modifier of the given block. */
BM: (block: string, ...modifiers: string[]) => string, 
/** @returns the class name for a modified element of the given block. */
BEM: (block: string, element: string, ...modifiers: string[]) => string, 
/** @returns the name of a custom property, including the leading dashes. */
variable: (...variables: string[]) => string;
/** Joins class values into a single class name, dropping the falsy ones. */
export declare const cls: (...classes: ClassValue[]) => string;
export declare const visible: (visible: boolean) => string;
export declare const expanded: (expanded: boolean) => string;
export declare const level: (level: text.Level) => string;
export declare const loc: (l: location.Crude) => string;
export declare const align: (position: spatial.Alignment | "") => string;
export declare const dir: (dir?: direction.Crude) => string | false;
export declare const disabled: (disabled?: boolean) => string | false;
export declare const bordered: (loc?: location.Crude | spatial.Alignment | boolean) => string | false;
export declare const noSelect: string;
export declare const selected: (selected: boolean) => string | false;
export declare const altColor: (secondary: boolean) => string | false;
export declare const editable: (editable: boolean) => string | false;
export declare const applyVars: typeof applyCSSVars;
export declare const removeVars: typeof removeCSSVars;
export declare const newGridBuilder: (prefix?: string) => CSSGridBuilder;
export declare const inheritDims: (inherit?: boolean) => string | false;
export declare const dropRegion: (active: boolean) => string | false;
export declare const px: (value: number) => string;
export declare function shade(value: Theming.Shade): string;
export declare function shade(value?: Theming.Shade): string | false;
export declare const colorVar: (value?: false | Theming.Shade | color.Crude) => string | undefined;
export declare const levelSizeVar: (value: string) => string;
/**
 * A style object that also accepts CSS custom properties. Use it in place of
 * `CSSProperties`, which rejects any `--` key.
 */
export type VarProperties = CSSProperties & Record<`--${string}`, string | number | undefined>;
//# sourceMappingURL=css.d.ts.map