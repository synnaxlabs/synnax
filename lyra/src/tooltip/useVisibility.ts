// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, TimeSpan } from "@synnaxlabs/x";
import { useEffect, useRef, useState } from "react";

import { useSyncedRef } from "@/hooks";
import { useConfig } from "@/tooltip/Config";
import { Triggers } from "@/triggers";

const CLOSE_DURATION = TimeSpan.milliseconds(150);
const ESCAPE_TRIGGERS: Triggers.Trigger[] = [Triggers.ESCAPE];

export interface UseVisibilityProps {
  /** Never opens while another tooltip is open. */
  yielding?: boolean;
}

export interface UseVisibilityReturn {
  /** The element the tooltip sits beside, or null while it is closed. */
  anchor: HTMLElement | null;
  /** True while the tooltip plays its exit. */
  closing: boolean;
  /** Opens beside the anchor after the hover delay, or at once while warm. */
  hover: (anchor: HTMLElement) => void;
  /** Opens beside the anchor at once. */
  open: (anchor: HTMLElement) => void;
  /** Closes the tooltip, skipping its exit when immediate. */
  close: (immediate?: boolean) => void;
}

/**
 * Runs the open and close cycle of one tooltip: the hover delay, the warm window, one
 * open tooltip at a time, and closing on Escape or when the anchor scrolls.
 */
export const useVisibility = ({
  yielding = false,
}: UseVisibilityProps = {}): UseVisibilityReturn => {
  const { delay, isWarm, isOpen, markClosed, acquire } = useConfig();
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const [closing, setClosing] = useState(false);
  const anchorRef = useRef<HTMLElement | null>(null);
  const openTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const releaseRef = useRef<destructor.Destructor | null>(null);
  const visibleRef = useSyncedRef(anchor != null);
  const delayRef = useSyncedRef(delay);

  const clearOpenTimeout = (): void => {
    if (openTimeoutRef.current == null) return;
    clearTimeout(openTimeoutRef.current);
    openTimeoutRef.current = null;
  };
  const clearCloseTimeout = (): void => {
    if (closeTimeoutRef.current == null) return;
    clearTimeout(closeTimeoutRef.current);
    closeTimeoutRef.current = null;
  };

  const close = (immediate: boolean = false): void => {
    clearOpenTimeout();
    if (!visibleRef.current) return;
    markClosed();
    releaseRef.current?.();
    releaseRef.current = null;
    if (immediate) {
      clearCloseTimeout();
      setClosing(false);
      setAnchor(null);
      return;
    }
    if (closeTimeoutRef.current != null) return;
    setClosing(true);
    closeTimeoutRef.current = setTimeout(() => {
      closeTimeoutRef.current = null;
      setClosing(false);
      setAnchor(null);
    }, CLOSE_DURATION.milliseconds);
  };
  const closeNow = (): void => close(true);

  const show = (): void => {
    releaseRef.current?.();
    releaseRef.current = null;
    if (yielding && isOpen()) return close(true);
    clearCloseTimeout();
    setClosing(false);
    releaseRef.current = acquire(closeNow);
    setAnchor(anchorRef.current);
  };

  const open = (next: HTMLElement): void => {
    clearOpenTimeout();
    anchorRef.current = next;
    show();
  };

  const hover = (next: HTMLElement): void => {
    clearOpenTimeout();
    anchorRef.current = next;
    if (visibleRef.current || isWarm()) return show();
    openTimeoutRef.current = setTimeout(
      show,
      new TimeSpan(delayRef.current).milliseconds,
    );
  };

  useEffect(() => {
    if (anchor == null) return;
    const handleScroll = (e: Event): void => {
      if (
        e.target instanceof Node &&
        (e.target === anchor || e.target.contains(anchor))
      )
        close(true);
    };
    window.addEventListener("scroll", handleScroll, { capture: true, passive: true });
    return () => window.removeEventListener("scroll", handleScroll, { capture: true });
  }, [anchor, close]);

  const handleEscape = ({ stage }: Triggers.UseEvent): void => {
    if (stage === "start" && visibleRef.current) close(true);
  };
  Triggers.use({ triggers: ESCAPE_TRIGGERS, callback: handleEscape, priority: 200 });

  useEffect(
    () => () => {
      clearOpenTimeout();
      clearCloseTimeout();
      releaseRef.current?.();
    },
    [clearOpenTimeout, clearCloseTimeout],
  );

  return { anchor, closing, hover, open, close };
};
