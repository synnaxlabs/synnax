import { ComponentType, ReactElement, cloneElement, useRef } from "react";

import { useSize } from "@/hooks";

/* AutoSize props is the props for the {@link AutoSize} component. */
export interface AutoSizeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  children: ComponentType<{ width: number; height: number }> | ReactElement;
  debounce?: number;
}

/**
 * AutoSize renders and tracks the dimensions of a div element and provides them to its
 * children.
 *
 * @param props - Standard div props that can be used to apply styles, classnames, etc.
 * to the parent div being measured.
 * @param props.debounce - An optional debounce time to set the maximum rate at which
 * AutoSize will rerender its children with updated dimensions. (A debounce time of
 * 100ms will mean that only one resize event will be propagated even if the component
 * is resized twice).
 */
export const AutoSize = ({
  children: Children,
  debounce,
  ...props
}: AutoSizeProps): JSX.Element => {
  const ref = useRef<HTMLDivElement>(null);
  const { width, height } = useSize({ ref, debounce });
  const content: ReactElement =
    typeof Children === "function" ? (
      <Children width={width} height={height} />
    ) : (
      cloneElement(Children, { width, height })
    );
  return (
    <div ref={ref} {...props}>
      {content}
    </div>
  );
};
