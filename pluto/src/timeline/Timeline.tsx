import "@/timeline/Timeline.css";

import { type bounds, box } from "@synnaxlabs/x";
import { type PropsWithChildren, useState } from "react";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Context, type ContextValue } from "@/timeline/context";

export interface FrameProps<T extends number | bigint = number> extends Flex.BoxProps {
  initialBounds: bounds.Bounds<T>;
}

export const Frame = <T extends number | bigint = number>({
  className,
  initialBounds,
  ...rest
}: FrameProps<T>) => {
  const [state, setState] = useState<ContextValue<T>>({
    bounds: initialBounds,
    setEntry: () => {},
    viewport: box.ZERO,
    setViewport: () => {},
  });
  return (
    <Context value={state as unknown as ContextValue}>
      <Flex.Box y className={CSS(CSS.BE("timeline", "frame"), className)} {...rest} />
    </Context>
  );
};
