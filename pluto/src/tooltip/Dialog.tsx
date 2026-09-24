// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, TimeSpan } from "@synnaxlabs/x";
import {
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  type Ref,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { useCombinedRefs, useSyncedRef } from "@/hooks";
import { type position } from "@/position";
import { Text } from "@/text";
import { useConfig } from "@/tooltip/Config";
import { Frame } from "@/tooltip/Frame";
import { Triggers } from "@/triggers";

interface ChildProps {
  ref?: Ref<HTMLElement>;
  "aria-describedby"?: string;
  "aria-label"?: string;
  children?: ReactNode;
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

/** The props a component adds to take an optional tooltip. */
export interface ExtensionProps {
  /** The tooltip content. Nothing shows while this is unset. */
  tooltip?: DialogProps["children"][0];
  /** The preferred location relative to the element. Chosen by position when unset. */
  tooltipLocation?: DialogProps["location"];
  /** Forces the tooltip to stay hidden. */
  hideTooltip?: DialogProps["hide"];
}

const CLOSE_DURATION = TimeSpan.milliseconds(150);
const ESCAPE_TRIGGERS: Triggers.Trigger[] = [Triggers.ESCAPE];

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

  const [anchor, setAnchor] = useState<HTMLElement | null>(null);
  const openTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const closeTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const releaseRef = useRef<destructor.Destructor | null>(null);
  const visibleRef = useSyncedRef(visible);
  const hideRef = useSyncedRef(hide);
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
      if (
        e.target instanceof Node &&
        anchor != null &&
        (e.target === anchor || e.target.contains(anchor))
      )
        close(true);
    };
    window.addEventListener("scroll", handleScroll, { capture: true, passive: true });
    return () => window.removeEventListener("scroll", handleScroll, { capture: true });
  }, [visible, close, anchor]);

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
  const combinedAnchorRef = useCombinedRefs(setAnchor, children_.props.ref);

  return (
    <>
      {visible && anchor != null && (
        <Frame
          id={id}
          anchor={anchor}
          location={locationProp}
          className={closing ? CSS.M("closing") : undefined}
          onAnchorLost={closeNow}
        >
          {formatTip(tip)}
        </Frame>
      )}
      {cloneElement(children_, {
        ref: combinedAnchorRef,
        "aria-describedby": visible ? id : children_.props["aria-describedby"],
        // A string tip doubles as an icon-only anchor's accessible name. An
        // anchor with visible text keeps that text as its name: aria-label
        // would override it, breaking label-in-name.
        "aria-label":
          children_.props["aria-label"] ??
          (typeof tip === "string" && Text.isSquare(children_.props.children)
            ? tip
            : undefined),
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
