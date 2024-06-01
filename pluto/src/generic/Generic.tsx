// Copyright 2024 Synnax Labs, Inc.
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
  type ForwardedRef,
  forwardRef,
  type ReactElement,
} from "react";

export type JSXElementType = keyof JSX.IntrinsicElements;

const CoreElement = <E extends JSXElementType>(
  { el, children, ...props }: ElementProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>,
): ReactElement => createElement(el, { ...props, ref }, children);

export type ElementProps<E extends JSXElementType> = ComponentPropsWithRef<E> & {
  el: E;
};

/**
 * Generic renders a component with the given element type .
 *
 * @param props - Props for the generic component. All props not defined below are passed to
 * the underlying element.
 * @param props.el - The element type to render.
 */
export const Element = forwardRef(CoreElement) as <E extends JSXElementType>(
  props: ElementProps<E>,
) => ReactElement;
