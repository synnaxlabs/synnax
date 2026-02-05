// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ComponentPropsWithRef,
  createElement,
  type JSX,
  memo,
  type ReactElement,
} from "react";

export type ElementType = keyof JSX.IntrinsicElements;

export type ElementPropsWithoutEl = ComponentPropsWithRef<ElementType>;

export type ElementProps<E extends keyof JSX.IntrinsicElements> = {
  el: E;
} & ComponentPropsWithRef<E>;

export type OptionalElementProps<E extends keyof JSX.IntrinsicElements> = {
  el?: E;
} & ComponentPropsWithRef<E>;

/**
 * Generic renders a component with the given element type .
 *
 * @param props - Props for the generic component. All props not defined below are passed to
 * the underlying element.
 * @param props.el - The element type to render.
 */
const BaseElement = <E extends keyof JSX.IntrinsicElements>({
  el,
  children,
  ...rest
}: ElementProps<E>): ReactElement => createElement(el, rest, children);

export const Element = memo(BaseElement) as typeof BaseElement;
