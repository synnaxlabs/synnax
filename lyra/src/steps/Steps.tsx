// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/steps/Steps.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { CSS } from "@/css";

/** The type scale steps a list can take. The markers scale with the text. */
export type Level = "p" | "small";

/** Props for {@link List}. */
export interface ListProps extends ComponentPropsWithoutRef<"ol"> {
  /** Type scale step for the items and their markers. Defaults to "p". */
  level?: Level;
}

/**
 * An ordered list of instructions. Each item gets a numbered square, and a line joins
 * the squares in order. Use it only when the order matters.
 *
 * @example
 * <Steps.List>
 *   <Steps.Item>Install the Driver.</Steps.Item>
 *   <Steps.Item>Plug in the device.</Steps.Item>
 * </Steps.List>
 */
export const List = ({ className, level = "p", ...rest }: ListProps): ReactElement => (
  <ol
    role="list"
    className={CSS.cls(CSS.B("steps"), CSS.BM("steps", level), className)}
    {...rest}
  />
);

/** Props for {@link Item}. */
export interface ItemProps extends ComponentPropsWithoutRef<"li"> {}

/** One step. It can hold a code block, a table, or an image under its text. */
export const Item = ({ className, ...rest }: ItemProps): ReactElement => (
  <li className={CSS.cls(CSS.BE("steps", "item"), className)} {...rest} />
);
