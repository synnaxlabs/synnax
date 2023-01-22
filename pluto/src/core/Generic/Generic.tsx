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
): JSX.Element => createElement(el, { ref, ...props }, children);

export type GenericProps<E extends ElementType> = ComponentPropsWithRef<E> & {
  el: E;
};

export const Generic = forwardRef(GenericCore) as <E extends JSXElementType>(
  props: GenericProps<E>
) => JSX.Element;
