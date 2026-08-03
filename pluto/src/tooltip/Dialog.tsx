// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/tooltip/Dialog.css";

import { box, type destructor, location, TimeSpan } from "@synnaxlabs/x";
import {
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  type Ref,
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { createPortal } from "react-dom";

import { CSS } from "@/css";
import { useCombinedRefs, useResize, useSyncedRef, useWindowResize } from "@/hooks";
import { position } from "@/position";
import { Text } from "@/text";
import { useConfig } from "@/tooltip/Config";
import { Triggers } from "@/triggers";
import { getRootElement } from "@/util/rootElement";

interface ChildProps {
  ref?: Ref<HTMLElement>;
  "aria-describedby"?: string;
  onPointerEnter?: (e: React.PointerEvent<HTMLElement>) => void;
  onPointerLeave?: (e: React.PointerEvent<HTMLElement>) => void;
  onPointerDown?: (e: React.PointerEvent<HTMLElement>) => void;
  onFocus?: (e: React.FocusEvent<HTMLElement>) => void;
  onBlur?: (e: React.FocusEvent<HTMLElement>) => void;
}

export interface DialogProps {
  location?: position.Location;
  hide?: boolean;
  children: [ReactNode, ReactElement<ChildProps>];
}

const PREFERENCES: position.LocationPreference[] = [
  { targetCorner: location.TOP_CENTER, dialogCorner: location.BOTTOM_CENTER },
  { targetCorner: location.BOTTOM_CENTER, dialogCorner: location.TOP_CENTER },
  { targetCorner: location.CENTER_RIGHT, dialogCorner: location.CENTER_LEFT },
  { targetCorner: location.CENTER_LEFT, dialogCorner: location.CENTER_RIGHT },
];

const OFFSET = 6;
const CLOSE_DURATION = TimeSpan.milliseconds(150);
const ESCAPE_TRIGGERS: Triggers.Trigger[] = [["Escape"]];

/**
 * A tooltip that appears when the user hovers or keyboard-focuses an element.
 *
 * @param props.children - The tooltip's content, followed by the element to attach
 * the tooltip to.
 * @param props.location - The preferred location for the tooltip relative to the
 * element. If unspecified or the tooltip would overflow the window, the best
 * location is chosen automatically.
 * @param props.hide - Force the tooltip to remain hidden.
 * @default false.
 */
export const Dialog = ({
  children,
  location: locationProp,
  hide = false,
}: DialogProps): ReactElement => {
  const { delay, isWarm, markClosed, acquire } = useConfig();
  const [visible, setVisible] = useState(false);
  const [closing, setClosing] = useState(false);
  const id = useId();

  const anchorRef = useRef<HTMLElement | null>(null);
  const tooltipRef = useRef<HTMLDivElement | null>(null);
  const openTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const releaseRef = useRef<destructor.Destructor | null>(null);
  const visibleRef = useSyncedRef(visible);
  const hideRef = useSyncedRef(hide);
  const locationRef = useSyncedRef(locationProp);
  const delayRef = useSyncedRef(delay);

  const clearOpenTimeout = useCallback((): void => {
    if (openTimeoutRef.current == null) return;
    clearTimeout(openTimeoutRef.current);
    openTimeoutRef.current = null;
  }, []);
  const clearCloseTimeout = useCallback((): void => {
    if (closeTimeoutRef.current == null) return;
    clearTimeout(closeTimeoutRef.current);
    closeTimeoutRef.current = null;
  }, []);

  const close = useCallback((immediate: boolean = false): void => {
    clearOpenTimeout();
    if (!visibleRef.current) return;
    markClosed();
    releaseRef.current?.();
    releaseRef.current = null;
    if (immediate) {
      clearCloseTimeout();
      setClosing(false);
      setVisible(false);
      return;
    }
    if (closeTimeoutRef.current != null) return;
    setClosing(true);
    closeTimeoutRef.current = setTimeout(() => {
      closeTimeoutRef.current = null;
      setClosing(false);
      setVisible(false);
    }, CLOSE_DURATION.milliseconds);
  }, []);
  const closeNow = useCallback((): void => close(true), [close]);

  const open = useCallback((): void => {
    clearCloseTimeout();
    setClosing(false);
    releaseRef.current?.();
    releaseRef.current = acquire(closeNow);
    setVisible(true);
  }, []);

  const positionTooltip = useCallback((): void => {
    const anchor = anchorRef.current;
    const el = tooltipRef.current;
    if (anchor == null || el == null) return;
    const target = box.construct(anchor);
    if (!anchor.isConnected || box.areaIsZero(target)) return close(true);
    const { adjustedDialog, dialogCorner } = position.position({
      target,
      dialog: box.construct(el),
      container: box.construct(0, 0, window.innerWidth, window.innerHeight),
      initial: locationRef.current,
      prefer: PREFERENCES,
      offset: OFFSET,
    });
    const rounded = box.round(adjustedDialog);
    el.style.left = CSS.px(box.left(rounded));
    el.style.top = CSS.px(box.top(rounded));
    el.style.transformOrigin = `${dialogCorner.x} ${dialogCorner.y}`;
  }, []);

  useLayoutEffect(() => {
    if (visible) positionTooltip();
  }, [visible, positionTooltip]);

  const resizeAnchorRef = useResize(positionTooltip, { enabled: visible });
  const resizeTooltipRef = useResize(positionTooltip, { enabled: visible });
  useWindowResize(positionTooltip, { enabled: visible });

  const handlePointerEnter = useCallback(
    (e: React.PointerEvent<HTMLElement>): void => {
      if (hideRef.current || e.pointerType === "touch") return;
      clearOpenTimeout();
      if (visibleRef.current || isWarm()) return open();
      openTimeoutRef.current = setTimeout(
        open,
        new TimeSpan(delayRef.current).milliseconds,
      );
    },
    [open],
  );
  const handlePointerLeave = useCallback((): void => close(), [close]);
  const handleFocus = useCallback(
    (e: React.FocusEvent<HTMLElement>): void => {
      if (hideRef.current || !e.currentTarget.matches(":focus-visible")) return;
      open();
    },
    [open],
  );

  useEffect(() => {
    if (!visible) return;
    const handleScroll = (e: Event): void => {
      const anchor = anchorRef.current;
      if (
        e.target instanceof Node &&
        anchor != null &&
        (e.target === anchor || e.target.contains(anchor))
      )
        close(true);
    };
    window.addEventListener("scroll", handleScroll, { capture: true, passive: true });
    return () => window.removeEventListener("scroll", handleScroll, { capture: true });
  }, [visible, close]);

  const handleEscape = useCallback(
    ({ stage }: Triggers.UseEvent): void => {
      if (stage === "start" && visibleRef.current) close(true);
    },
    [close],
  );
  Triggers.use({ triggers: ESCAPE_TRIGGERS, callback: handleEscape, priority: 200 });

  useEffect(() => {
    if (hide) close(true);
  }, [hide, close]);

  useEffect(
    () => () => {
      clearOpenTimeout();
      clearCloseTimeout();
      releaseRef.current?.();
    },
    [],
  );

  const [tip, children_] = children;
  const combinedAnchorRef = useCombinedRefs(
    anchorRef,
    resizeAnchorRef,
    children_.props.ref,
  );
  const combinedTooltipRef = useCombinedRefs(tooltipRef, resizeTooltipRef);

  return (
    <>
      {visible &&
        createPortal(
          <div
            id={id}
            role="tooltip"
            ref={combinedTooltipRef}
            className={CSS(CSS.B("tooltip"), closing && CSS.M("closing"))}
          >
            {formatTip(tip)}
          </div>,
          getRootElement(),
        )}
      {cloneElement(children_, {
        ref: combinedAnchorRef,
        "aria-describedby": visible ? id : children_.props["aria-describedby"],
        onPointerEnter: (e) => {
          handlePointerEnter(e);
          children_.props.onPointerEnter?.(e);
        },
        onPointerLeave: (e) => {
          handlePointerLeave();
          children_.props.onPointerLeave?.(e);
        },
        onPointerDown: (e) => {
          closeNow();
          children_.props.onPointerDown?.(e);
        },
        onFocus: (e) => {
          handleFocus(e);
          children_.props.onFocus?.(e);
        },
        onBlur: (e) => {
          closeNow();
          children_.props.onBlur?.(e);
        },
      })}
    </>
  );
};

export const formatTip = (tip: ReactNode): ReactNode => {
  if (typeof tip === "string" || typeof tip === "number" || !isValidElement(tip))
    return (
      <Text.Text level="small" color={11}>
        {tip as string | number}
      </Text.Text>
    );
  return tip;
};
