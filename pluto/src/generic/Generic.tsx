// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ComponentPropsWithRef,
  createElement,
  ElementType,
  ForwardedRef,
  forwardRef,
  ReactElement,
} from "react";

export type JSXElementType = keyof JSX.IntrinsicElements;

const Core = <E extends JSXElementType>(
  { el, children, ...props }: GenericProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>
): ReactElement => createElement(el, { ...props, ref }, children);

export type GenericProps<E extends ElementType> = ComponentPropsWithRef<E> & {
  el: E;
};

/**
 * Generic renders a component with the given element type .
 *
 * @param props - Props for the generic component. All props not defined below are passed to
 * the underlying element.
 * @param props.el - The element type to render.
 */
export const Generic = forwardRef(Core) as <E extends JSXElementType>(
  props: GenericProps<E>
) => ReactElement;
