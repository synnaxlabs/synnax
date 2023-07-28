// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useCallback,
  useContext,
  useRef,
  useState,
} from "react";

import { TimeSpan, CrudeTimeSpan } from "@synnaxlabs/x";

export interface TooltipContextValue {
  delay: CrudeTimeSpan;
  startAccelerating: () => void;
}

export interface TooltipConfigProps
  extends PropsWithChildren,
    Partial<Omit<TooltipContextValue, "startAccelerating">> {
  accelerate?: boolean;
  acceleratedDelay?: CrudeTimeSpan;
  accelartionDuration?: CrudeTimeSpan;
}

const ZERO_TOOLTIP_CONFIG: TooltipContextValue = {
  delay: TimeSpan.milliseconds(50),
  startAccelerating: () => {},
};

export const TooltipContext = createContext<TooltipContextValue>(ZERO_TOOLTIP_CONFIG);

export const useTooltipConfig = (): TooltipContextValue => useContext(TooltipContext);

export const TooltipConfig = ({
  delay = TimeSpan.milliseconds(500),
  accelerate = true,
  acceleratedDelay = TimeSpan.milliseconds(50),
  accelartionDuration = TimeSpan.seconds(5),
  children,
}: TooltipConfigProps): ReactElement => {
  const [accelerating, setAccelerating] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startAccelerating = useCallback((): void => {
    if (accelerating || !accelerate) return;
    setAccelerating(true);
    timeoutRef.current = setTimeout(() => {
      setAccelerating(false);
    }, new TimeSpan(accelartionDuration).milliseconds);
  }, [accelerating, accelartionDuration]);
  return (
    <TooltipContext.Provider
      value={{
        delay: accelerating ? acceleratedDelay : delay,
        startAccelerating,
      }}
    >
      {children}
    </TooltipContext.Provider>
  );
};
