import {
  createElement,
  DetailedHTMLProps,
  ForwardedRef,
  forwardRef,
  HtmlHTMLAttributes,
} from "react";

export type GenericElementKey<P extends keyof JSX.IntrinsicElements = any> = keyof Pick<
  JSX.IntrinsicElements,
  P
>;

export interface GenericProps<E extends HTMLElement = HTMLElement>
  extends DetailedHTMLProps<HtmlHTMLAttributes<E>, E> {
  el: GenericElementKey;
}

const GenericCore = <E extends HTMLElement = HTMLElement>(
  { el = "h1", children, ...props }: GenericProps<E>,
  ref: ForwardedRef<E>
): JSX.Element => createElement(el, { ref, ...props }, children);

export const Generic = forwardRef(GenericCore) as <E extends HTMLElement = HTMLElement>(
  props: GenericProps<E> & { ref?: ForwardedRef<E> }
) => JSX.Element;
