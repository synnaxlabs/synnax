import { cloneElement, ComponentType, ReactElement, useRef } from "react";
import { useResize, useResizeOpts } from "../../Hooks";

export interface AutoSizeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children">,
    useResizeOpts {
  children: ComponentType<{ width: number; height: number }> | ReactElement;
}

export default function AutoSize({
  children: Children,
  style,
  debounce,
  ...props
}: AutoSizeProps) {
  const ref = useRef<HTMLDivElement>(null);
  const { width, height } = useResize(ref, { debounce });
  let content: ReactElement =
    typeof Children === "function" ? (
      <Children width={width} height={height} />
    ) : (
      cloneElement(Children, { width, height })
    );
  return (
    <div ref={ref} style={style} {...props}>
      {content}
    </div>
  );
}
