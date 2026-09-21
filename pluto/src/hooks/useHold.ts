// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import {
  type KeyboardEventHandler,
  type MouseEventHandler,
  useEffect,
  useMemo,
  useState,
} from "react";

import { useDestructors } from "@/hooks/useDestructors";
import { ACTIVATION_KEYS } from "@/util/event";

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
  /** Whether a primary-button or activation-key press is in progress. */
  pressed: boolean;
  onClick: MouseEventHandler<E>;
  onMouseDown: MouseEventHandler<E>;
  /** Paints the pressed state for Space and Enter. Never starts a hold. */
  onKeyDown: KeyboardEventHandler<E>;
  onKeyUp: KeyboardEventHandler<E>;
}

// A native drag and a window that loses focus both swallow the mouseup.
const releaseTargets = (): [EventTarget, string][] => [
  [document, "mouseup"],
  [document, "dragstart"],
  [window, "blur"],
];

/**
 * Gates onClick behind a press-and-hold of onClickDelay. Only a primary press starts
 * the hold. A release, drag, window blur, unmount, or disable cancels it.
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
    if (disabled || e.button !== 0) return;
    destructors.cleanup();
    setPressed(true);
    const timeout = delay.isZero
      ? null
      : setTimeout(() => onClick?.(e), delay.milliseconds);
    const targets = releaseTargets();
    const release = (): void => {
      setPressed(false);
      if (timeout != null) clearTimeout(timeout);
      for (const [target, ev] of targets) target.removeEventListener(ev, release);
    };
    for (const [target, ev] of targets) target.addEventListener(ev, release);
    destructors.set(release);
  };

  const handleKeyDown: KeyboardEventHandler<E> = (e) => {
    if (disabled || !delay.isZero || e.defaultPrevented || e.repeat) return;
    if (e.target !== e.currentTarget || !ACTIVATION_KEYS.includes(e.key)) return;
    setPressed(true);
  };

  const handleKeyUp: KeyboardEventHandler<E> = (e) => {
    if (ACTIVATION_KEYS.includes(e.key)) setPressed(false);
  };

  return {
    delay,
    pressed,
    onClick: handleClick,
    onMouseDown: handleMouseDown,
    onKeyDown: handleKeyDown,
    onKeyUp: handleKeyUp,
  };
};
