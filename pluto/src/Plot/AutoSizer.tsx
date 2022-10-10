import { ComponentType, useRef } from "react";
import { useResize, useResizeOpts } from "../util/useResize";

export interface AutoSizerProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children">,
    useResizeOpts {
  children: ComponentType<{ width: number; height: number }>;
}

export default function AutoSizer({
  children: Children,
  style,
  debounce,
  ...props
}: AutoSizerProps) {
  const ref = useRef<HTMLDivElement>(null);
  const { width, height } = useResize(ref, { debounce });
  return (
    <div ref={ref} style={{ ...style }} {...props}>
      <Children width={width} height={height} />
    </div>
  );
}
