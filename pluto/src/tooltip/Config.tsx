// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useMemo,
  useRef,
  useState,
} from "react";

export interface ContextValue {
  delay: CrudeTimeSpan;
  startAccelerating: () => void;
}

export interface ConfigProps
  extends PropsWithChildren,
    Partial<Omit<ContextValue, "startAccelerating">> {
  accelerate?: boolean;
  acceleratedDelay?: CrudeTimeSpan;
  accelerationDelay?: CrudeTimeSpan;
}

const Context = createContext<ContextValue>({
  delay: TimeSpan.milliseconds(750),
  startAccelerating: () => {},
});
Context.displayName = "Tooltip.Context";

export const useConfig = () => use(Context);

/**
 * Sets the default configuration for all tooltips in its children.
 *
 * @param props - The props for the tooltip config.
 * @param props.delay - The delay before the tooltip appears, in milliseconds.
 * @default 500ms.
 * @param props.accelerate - Whether to enable accelerated visibility of tooltips for a
 * short period of time after the user has hovered over a first tooltip.
 * @default true.
 * @param props.acceleratedDelay - The delay before the tooltip appears when
 * accelerated visibility is enabled.
 * @default 100 ms.
 * @param props.acceleratedDuration - The duration of accelerated visibility.
 * @default 10 seconds.
 */
export const Config = ({
  delay = TimeSpan.milliseconds(700),
  accelerate = true,
  // Disabling this for now because it's annoying.
  acceleratedDelay = TimeSpan.minutes(60),
  accelerationDelay = TimeSpan.seconds(0),
  children,
}: ConfigProps): ReactElement => {
  const [accelerating, setAccelerating] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startAccelerating = useCallback((): void => {
    if (accelerating || !accelerate) return;
    setAccelerating(true);
    timeoutRef.current = setTimeout(() => {
      setAccelerating(false);
    }, new TimeSpan(accelerationDelay).milliseconds);
  }, [accelerating, accelerationDelay]);
  const parsedDelay = useMemo(
    () => (accelerating ? acceleratedDelay : delay),
    [accelerating, acceleratedDelay, delay],
  );
  const value = useMemo(
    () => ({ delay: parsedDelay, startAccelerating }),
    [parsedDelay, startAccelerating],
  );
  return <Context value={value}>{children}</Context>;
};
