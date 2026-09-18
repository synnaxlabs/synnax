// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type MouseEventHandler, useEffect, useRef, useState } from "react";

export interface UseHoldProps<E extends Element> {
  onClick?: MouseEventHandler<E>;
  onMouseDown?: MouseEventHandler<E>;
  onClickDelay?: CrudeTimeSpan;
  /** Ignores presses and cancels a hold in progress. */
  disabled?: boolean;
}

export interface UseHoldReturn<E extends Element> {
  /** The activation delay. Zero means a click actuates at once. */
  delay: TimeSpan;
  /** Whether a primary-button press is in progress. */
  pressed: boolean;
  onClick: MouseEventHandler<E>;
  onMouseDown: MouseEventHandler<E>;
}

/**
 * Gates onClick behind a press-and-hold of onClickDelay. A primary press starts the
 * hold, and a release before the delay cancels it. Secondary buttons never actuate.
 * Unmounting or disabling the control cancels a hold in progress.
 */
export const useHold = <E extends Element>({
  onClick,
  onMouseDown,
  onClickDelay = 0,
  disabled = false,
}: UseHoldProps<E>): UseHoldReturn<E> => {
  const delay = TimeSpan.fromMilliseconds(onClickDelay);
  const cancelRef = useRef<(() => void) | null>(null);
  // WebKit sets :active on a secondary press, so pressed styling follows this flag.
  const [pressed, setPressed] = useState(false);

  useEffect(() => {
    if (disabled) cancelRef.current?.();
    return () => cancelRef.current?.();
  }, [disabled]);

  const handleClick: MouseEventHandler<E> = (e) => {
    if (delay.isZero) onClick?.(e);
  };

  const handleMouseDown: MouseEventHandler<E> = (e) => {
    onMouseDown?.(e);
    if (disabled || e.button !== 0) return;
    cancelRef.current?.();
    setPressed(true);
    const timeout = delay.isZero
      ? null
      : setTimeout(() => {
          cancelRef.current = null;
          onClick?.(e);
        }, delay.milliseconds);
    const release = (): void => {
      cancelRef.current = null;
      setPressed(false);
      if (timeout != null) clearTimeout(timeout);
    };
    document.addEventListener("mouseup", release, { once: true });
    cancelRef.current = () => {
      document.removeEventListener("mouseup", release);
      release();
    };
  };

  return { delay, pressed, onClick: handleClick, onMouseDown: handleMouseDown };
};
