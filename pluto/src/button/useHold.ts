// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type MouseEventHandler, useEffect, useMemo, useState } from "react";

import { useDestructors } from "@/hooks";
import { Triggers } from "@/triggers";

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

// A native drag swallows the mouseup, so dragstart releases the hold too.
const RELEASE_EVENTS = ["mouseup", "dragstart"];

/**
 * Gates onClick behind a press-and-hold of onClickDelay. A primary press starts the
 * hold, and a release before the delay cancels it. Secondary buttons never actuate. A
 * drag, unmounting, or disabling the control releases a hold in progress.
 */
export const useHold = <E extends Element>({
  onClick,
  onMouseDown,
  onClickDelay = 0,
  disabled = false,
}: UseHoldProps<E>): UseHoldReturn<E> => {
  const delay = useMemo(() => TimeSpan.fromMilliseconds(onClickDelay), [onClickDelay]);
  const destructors = useDestructors();
  // WebKit sets :active on a secondary press, so pressed styling follows this flag.
  const [pressed, setPressed] = useState(false);

  useEffect(() => destructors.cleanup, [disabled, destructors]);

  const handleClick: MouseEventHandler<E> = (e) => {
    if (delay.isZero) onClick?.(e);
  };

  const handleMouseDown: MouseEventHandler<E> = (e) => {
    onMouseDown?.(e);
    if (disabled || e.button !== Triggers.MOUSE_LEFT_NUMBER) return;
    destructors.cleanup();
    setPressed(true);
    const timeout = delay.isZero
      ? null
      : setTimeout(() => onClick?.(e), delay.milliseconds);
    const release = (): void => {
      setPressed(false);
      if (timeout != null) clearTimeout(timeout);
      for (const ev of RELEASE_EVENTS) document.removeEventListener(ev, release);
    };
    for (const ev of RELEASE_EVENTS) document.addEventListener(ev, release);
    destructors.set(release);
  };

  return { delay, pressed, onClick: handleClick, onMouseDown: handleMouseDown };
};
