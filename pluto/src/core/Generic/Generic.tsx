import {
  ComponentPropsWithRef,
  createElement,
  ElementType,
  ForwardedRef,
  forwardRef,
} from "react";

export type JSXElementType = keyof JSX.IntrinsicElements;

const GenericCore = <E extends JSXElementType>(
  { el, children, ...props }: GenericProps<E>,
  ref: ForwardedRef<JSX.IntrinsicElements[E]>
): JSX.Element => createElement(el, { ...props, ref }, children);

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
export const Generic = forwardRef(GenericCore) as <E extends JSXElementType>(
  props: GenericProps<E>
) => JSX.Element;
