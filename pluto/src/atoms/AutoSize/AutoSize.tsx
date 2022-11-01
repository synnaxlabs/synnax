import { ComponentType, ReactElement, cloneElement, useRef } from "react";
import { useSize } from "@/hooks";

export interface AutoSizeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  children: ComponentType<{ width: number; height: number }> | ReactElement;
  debounce?: number;
}
export const AutoSize = ({
  children: Children,
  debounce,
  ...props
}: AutoSizeProps) => {
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
