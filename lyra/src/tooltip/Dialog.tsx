// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  cloneElement,
  isValidElement,
  type ReactElement,
  type ReactNode,
  type Ref,
  useEffect,
  useId,
} from "react";

import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { type position } from "@/position";
import { Text } from "@/text";
import { Frame } from "@/tooltip/Frame";
import { useVisibility } from "@/tooltip/useVisibility";

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
  /** Keeps the tooltip hidden. A function decides on each hover or focus. */
  hide?: boolean | ((anchor: HTMLElement) => boolean);
  children: [ReactNode, ReactElement<ChildProps>];
}

/** The props a component adds to take an optional tooltip. */
export interface ExtensionProps {
  /** The tooltip content. Nothing shows while this is unset. */
  tooltip?: DialogProps["children"][0];
  /** The preferred location relative to the element. Chosen by position when unset. */
  tooltipLocation?: DialogProps["location"];
  /** Keeps the tooltip hidden. A function decides on each hover or focus. */
  hideTooltip?: DialogProps["hide"];
}

interface AnchorProps extends ChildProps {
  element: ReactElement<ChildProps>;
}

// Clones in its own component because the compiler cannot tell that cloneElement never
// calls the handlers it receives, while it knows render never calls a JSX prop.
const Anchor = ({ element, ...props }: AnchorProps): ReactElement =>
  cloneElement(element, props);

/**
 * A tooltip that appears when the user hovers or keyboard-focuses an element.
 *
 * @param props.children - The tooltip's content, followed by the element to attach
 * the tooltip to.
 * @param props.location - The preferred location for the tooltip relative to the
 * element. If unspecified or the tooltip would overflow the window, the best
 * location is chosen automatically.
 * @param props.hide - Force the tooltip to remain hidden, or decide on each hover or
 * focus from the anchor.
 * @default false.
 */
export const Dialog = ({
  children,
  location: locationProp,
  hide = false,
}: DialogProps): ReactElement => {
  const { anchor, closing, hover, open, close } = useVisibility();
  const visible = anchor != null;
  const id = useId();
  const hideRef = useSyncedRef(hide);
  const hidden = (anchor: HTMLElement): boolean => {
    const h = hideRef.current;
    return typeof h === "function" ? h(anchor) : h;
  };
  const closeNow = (): void => close(true);

  const handlePointerEnter = (e: React.PointerEvent<HTMLElement>): void => {
    if (e.pointerType === "touch" || hidden(e.currentTarget)) return;
    hover(e.currentTarget);
  };
  const handlePointerLeave = (): void => close();
  const handleFocus = (e: React.FocusEvent<HTMLElement>): void => {
    if (!e.currentTarget.matches(":focus-visible") || hidden(e.currentTarget)) return;
    open(e.currentTarget);
  };

  useEffect(() => {
    if (hide === true) close(true);
  }, [hide, close]);

  const [tip, children_] = children;

  return (
    <>
      {anchor != null && (
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
      <Anchor
        element={children_}
        aria-describedby={visible ? id : children_.props["aria-describedby"]}
        // A string tip doubles as an icon-only anchor's accessible name. An anchor with
        // visible text keeps that text as its name: aria-label would override it,
        // breaking label-in-name.
        aria-label={
          children_.props["aria-label"] ??
          (typeof tip === "string" && Text.isSquare(children_.props.children)
            ? tip
            : undefined)
        }
        onPointerEnter={(e) => {
          handlePointerEnter(e);
          children_.props.onPointerEnter?.(e);
        }}
        onPointerLeave={(e) => {
          handlePointerLeave();
          children_.props.onPointerLeave?.(e);
        }}
        onPointerDown={(e) => {
          closeNow();
          children_.props.onPointerDown?.(e);
        }}
        onFocus={(e) => {
          handleFocus(e);
          children_.props.onFocus?.(e);
        }}
        onBlur={(e) => {
          closeNow();
          children_.props.onBlur?.(e);
        }}
      />
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
