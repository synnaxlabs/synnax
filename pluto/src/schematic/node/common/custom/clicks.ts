// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { useHold } from "@synnaxlabs/lyra/hooks";
import { useMemoArray } from "@synnaxlabs/lyra/memo";
import type { CrudeTimeSpan } from "@synnaxlabs/x";
import {
  type CSSProperties,
  type MouseEventHandler,
  type RefObject,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";
import { Region } from "@/schematic/node/common/region";

const CLICKABLE = CSS.BEM("symbol", "part", "clickable");
const HOVERED = CSS.BEM("symbol", "part", "hovered");
const PRESSED = CSS.BEM("symbol", "part", "pressed");
const OPEN = CSS.BEM("symbol", "part", "open");

// partAt returns the key of the part that holds target inside host.
const partAt = (
  parts: readonly schematic.symbol.Part[],
  host: Element | null,
  target: EventTarget | null,
): string | undefined => {
  if (host == null || !(target instanceof Element)) return undefined;
  return parts.find(({ selectors }) =>
    selectors.some((s) => {
      const el = target.closest(s);
      return el != null && host.contains(el);
    }),
  )?.key;
};

// mark gives className to each of shapes inside host, and to no other element.
const mark = (host: Element, shapes: SVGElement[], className: string) => {
  host
    .querySelectorAll(`.${className}`)
    .forEach((el) => el.classList.remove(className));
  shapes.forEach((el) => el.classList.add(className));
};

// SVG fills an open shape as if it were closed, so an open shape takes no tint fill.
const isOpen = (el: SVGElement): boolean =>
  el.tagName === "line" ||
  el.tagName === "polyline" ||
  (el.tagName === "path" &&
    (el.getAttribute("d") ?? "")
      .split(/(?=[Mm])/)
      .some((sub) => sub.trim() !== "" && !/[Zz]\s*$/.test(sub)));

// A press released off its part makes the browser click the shapes around both ends.
// dropStrayClick drops that click until the press ends, so no other part and not the
// drawing takes it.
const dropStrayClick = (
  doc: Document,
  onPart: (target: EventTarget | null) => boolean,
) => {
  const drop = (e: MouseEvent) => {
    if (!onPart(e.target)) e.stopPropagation();
  };
  const end = () => {
    doc.removeEventListener("click", drop, true);
    doc.removeEventListener("mouseup", afterRelease, true);
    doc.removeEventListener("dragstart", end, true);
    doc.defaultView?.removeEventListener("blur", end);
  };
  // The click comes after the mouseup in the same task.
  const afterRelease = () => setTimeout(end);
  doc.addEventListener("click", drop, true);
  doc.addEventListener("mouseup", afterRelease, true);
  doc.addEventListener("dragstart", end, true);
  doc.defaultView?.addEventListener("blur", end);
};

export interface UsePartClicksParams {
  spec: schematic.symbol.Spec;
  /** clicks holds what a click on each part does, by part key. */
  clicks: Record<string, () => void>;
  /** delay is how long a press on a part lasts before its click runs. */
  delay?: CrudeTimeSpan;
}

export interface UsePartClicksReturn {
  /** ref goes on the element that holds the drawing. */
  ref: RefObject<SVGGElement | null>;
  className?: string;
  style?: CSSProperties;
  onMouseDown: MouseEventHandler<SVGGElement>;
  onClick: MouseEventHandler<SVGGElement>;
  onMouseOver: MouseEventHandler<SVGGElement>;
  onMouseLeave: MouseEventHandler<SVGGElement>;
}

/**
 * Makes each part of a drawing that has a click into its own button. Attach the return
 * to the element that holds the drawing. A press on a part stops there, so the button
 * around the drawing does not take it. The part tints on hover and press.
 */
export const usePartClicks = ({
  spec,
  clicks,
  delay,
}: UsePartClicksParams): UsePartClicksReturn => {
  const hostRef = useRef<SVGGElement | null>(null);
  // Clicks change on each render of a node, so only a change of parts marks the shapes.
  const clickable = useMemoArray(
    spec.parts.filter(({ key }) => Object.hasOwn(clicks, key)),
  );
  const at = (target: EventTarget | null) => partAt(clickable, hostRef.current, target);
  const [hovered, setHovered] = useState<string>();
  const [pressed, setPressed] = useState<string>();
  const hold = useHold<SVGGElement>({
    onClick: ({ target }) => {
      const key = at(target);
      if (key != null) clicks[key]();
    },
    onClickDelay: delay,
  });
  const shownPress = hold.pressed ? pressed : undefined;
  // A new markup mounts a new drawing, which has no marks.
  useLayoutEffect(() => {
    const host = hostRef.current;
    if (host == null) return;
    const shapes = (parts: readonly schematic.symbol.Part[]) =>
      parts.flatMap(({ selectors }) => Region.graphics(host, selectors));
    const keyed = (key?: string) => shapes(clickable.filter((p) => p.key === key));
    const all = shapes(clickable);
    mark(host, all, CLICKABLE);
    mark(host, all.filter(isOpen), OPEN);
    mark(host, keyed(hovered), HOVERED);
    mark(host, keyed(shownPress), PRESSED);
  }, [spec.drawing.markup, clickable, hovered, shownPress]);
  const delayed = !hold.delay.isZero;
  return {
    ref: hostRef,
    className: delayed ? CSS.BM("symbol-parts", "delayed") : undefined,
    style: delayed
      ? { [Primitive.TOGGLE_DELAY]: `${hold.delay.seconds.toString()}s` }
      : undefined,
    onMouseDown: (e) => {
      const key = at(e.target);
      if (key == null) return;
      e.stopPropagation();
      setPressed(key);
      hold.onMouseDown(e);
      if (e.button === 0)
        dropStrayClick(e.currentTarget.ownerDocument, (t) => at(t) === key);
    },
    onClick: (e) => {
      if (at(e.target) == null) return;
      e.stopPropagation();
      hold.onClick(e);
    },
    onMouseOver: (e) => setHovered(at(e.target)),
    onMouseLeave: () => setHovered(undefined),
  };
};
