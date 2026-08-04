// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CrudeTimeSpan,
  type destructor,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { context } from "@/context";

export interface ContextValue {
  delay: CrudeTimeSpan;
  /** Returns true if a tooltip closed recently enough to skip the open delay. */
  isWarm: () => boolean;
  /** Records a tooltip close, starting the warm window. */
  markClosed: () => void;
  /**
   * Registers the currently open tooltip's close function, closing any other open
   * tooltip. Returns a destructor that deregisters the tooltip.
   */
  acquire: (close: () => void) => destructor.Destructor;
}

export interface ConfigProps extends PropsWithChildren {
  delay?: CrudeTimeSpan;
  skipDelay?: CrudeTimeSpan;
}

const [Context, useConfig] = context.create<ContextValue>({
  defaultValue: {
    delay: TimeSpan.milliseconds(700),
    isWarm: () => false,
    markClosed: () => {},
    acquire: () => () => {},
  },
  displayName: "Tooltip.Context",
});
export { useConfig };

/**
 * Sets the configuration for all tooltips in its children.
 *
 * @param props - The props for the tooltip config.
 * @param props.delay - The delay before a tooltip opens on hover.
 * @default 700ms.
 * @param props.skipDelay - How long after a tooltip closes a new hover opens
 * instantly.
 * @default 300ms.
 */
export const Config = ({
  delay = TimeSpan.milliseconds(700),
  skipDelay = TimeSpan.milliseconds(300),
  children,
}: ConfigProps): ReactElement => {
  const lastClosed = useRef<TimeStamp | null>(null);
  const open = useRef<(() => void) | null>(null);
  const skipDelayRef = useRef(skipDelay);
  skipDelayRef.current = skipDelay;

  const isWarm = useCallback((): boolean => {
    if (lastClosed.current == null) return false;
    const since = TimeStamp.since(lastClosed.current);
    return since.valueOf() < new TimeSpan(skipDelayRef.current).valueOf();
  }, []);
  const markClosed = useCallback((): void => {
    lastClosed.current = TimeStamp.now();
  }, []);
  const acquire = useCallback((close: () => void): destructor.Destructor => {
    if (open.current !== close) open.current?.();
    open.current = close;
    return () => {
      if (open.current === close) open.current = null;
    };
  }, []);

  const value = useMemo<ContextValue>(
    () => ({ delay, isWarm, markClosed, acquire }),
    [delay, isWarm, markClosed, acquire],
  );
  return <Context value={value}>{children}</Context>;
};
