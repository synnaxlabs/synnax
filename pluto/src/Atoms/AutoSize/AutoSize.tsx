import {
  cloneElement,
  ComponentType,
  ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";
import { useResize, useResizeOpts } from "../../Hooks";

export interface AutoSizeProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "children">,
    useResizeOpts {
  children: ComponentType<{ width: number; height: number }> | ReactElement;
}
const AutoSize = ({
  children: Children,
  debounce,
  ...props
}: AutoSizeProps) => {
  const ref = useRef<HTMLDivElement>(null);
  const { width, height } = useResize(ref, { debounce });
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
