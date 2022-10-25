import {
  cloneElement,
  ComponentType,
  ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";
import { useSize } from "../../Hooks/useResize";

export interface AutoSizeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children"> {
  children: ComponentType<{ width: number; height: number }> | ReactElement;
  debounce?: number;
}
const AutoSize = ({
  children: Children,
  debounce,
  ...props
}: AutoSizeProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const { width, height } = useSize({ ref, debounce });
  let content: ReactElement =
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

export default AutoSize;
