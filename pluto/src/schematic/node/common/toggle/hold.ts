// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type MouseEventHandler, useRef, useState } from "react";

export interface UseHoldProps<E extends Element> {
  onClick?: MouseEventHandler<E>;
  onMouseDown?: MouseEventHandler<E>;
  onClickDelay?: CrudeTimeSpan;
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
 */
export const useHold = <E extends Element>({
  onClick,
  onMouseDown,
  onClickDelay = 0,
}: UseHoldProps<E>): UseHoldReturn<E> => {
  const delay = TimeSpan.fromMilliseconds(onClickDelay);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // WebKit sets :active on a secondary press, so pressed styling follows this flag.
  const [pressed, setPressed] = useState(false);

  const handleClick: MouseEventHandler<E> = (e) => {
    if (delay.isZero) onClick?.(e);
  };

  const handleMouseDown: MouseEventHandler<E> = (e) => {
    onMouseDown?.(e);
    if (e.button !== 0) return;
    setPressed(true);
    document.addEventListener(
      "mouseup",
      () => {
        setPressed(false);
        if (timeoutRef.current != null) clearTimeout(timeoutRef.current);
        timeoutRef.current = null;
      },
      { once: true },
    );
    if (delay.isZero) return;
    timeoutRef.current = setTimeout(() => {
      onClick?.(e);
      timeoutRef.current = null;
    }, delay.milliseconds);
  };

  return { delay, pressed, onClick: handleClick, onMouseDown: handleMouseDown };
};
