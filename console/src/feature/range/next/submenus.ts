// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dialog } from "@synnaxlabs/lyra/dialog";
import { state as xstate, type xy } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type MouseEvent,
  type PointerEvent,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";

export type MenuProperty = "labels" | "stage" | "time" | "parent";

/** Names the property of a trigger in the menu. */
const PROPERTY_ATTRIBUTE = "data-property";

/** How long the pointer rests on a property before its values open. */
const REST_MS = 100;
/** How long the pointer may travel toward open values while they stay open. */
const AIM_MS = 300;
/** How far behind the pointer the aim area starts, so a small jitter stays inside. */
const AIM_BLEED = 5;
/** How far the pointer must move toward the values to count as travel to them. */
const AIM_MIN_TRAVEL = 2;
/** How far the pointer may drift up or down while it holds still. */
const AIM_MAX_DRIFT = 3;

interface Aim {
  polygon: xy.XY[];
  expires: number;
  /** Where the pointer left the property. */
  exit: xy.XY;
  /** Whether the values open to the right of the menu. */
  right: boolean;
}

const contains = (polygon: xy.XY[], p: xy.XY): boolean => {
  let inside = false;
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const a = polygon[i];
    const b = polygon[j];
    if (
      a.y > p.y !== b.y > p.y &&
      p.x < ((b.x - a.x) * (p.y - a.y)) / (b.y - a.y) + a.x
    )
      inside = !inside;
  }
  return inside;
};

const propertyOf = (target: EventTarget): MenuProperty | null => {
  if (!(target instanceof Element)) return null;
  const el = target.closest(`[${PROPERTY_ATTRIBUTE}]`);
  return (el?.getAttribute(PROPERTY_ATTRIBUTE) as MenuProperty | null) ?? null;
};

interface State {
  open: MenuProperty | null;
  /** Whether the open values replaced other open values. */
  swap: boolean;
}

const transition = (prev: State, open: MenuProperty | null): State => {
  if (prev.open === open) return prev;
  return { open, swap: prev.open != null && open != null };
};

export interface SurfaceHandlers {
  onClick: (e: MouseEvent<HTMLElement>) => void;
  onPointerMove: (e: PointerEvent<HTMLElement>) => void;
  onPointerLeave: (e: PointerEvent<HTMLElement>) => void;
  onKeyDownCapture: (e: KeyboardEvent<HTMLElement>) => void;
}

/** Names the surface of the values of a property. */
const SURFACE_ATTRIBUTE = "data-surface";

export interface Submenus {
  /** Whether the pointer travels toward open values. */
  aiming: boolean;
  /** Spread on the element that holds the properties. Their values bubble to it too. */
  surface: SurfaceHandlers;
  frame: (
    property: MenuProperty,
  ) => Pick<Dialog.FrameProps, "visible" | "onVisibleChange">;
  picker: (property: MenuProperty) => {
    surfaceID: string;
    className: string;
  };
}

/**
 * Opens the values of one property of a menu at a time, beside the menu. A rest on a
 * property opens its values, a click or ArrowRight opens them at once, and ArrowLeft
 * or Escape closes them. While the pointer moves toward open values, the properties it
 * crosses neither light nor open. Each trigger names its property with a
 * `data-property` attribute, and one set of handlers on the menu reads them all.
 */
export const useSubmenus = (): Submenus => {
  const [{ open, swap }, setState] = useState<State>({ open: null, swap: false });
  const [aiming, setAiming] = useState(false);
  const restTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const aimTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const aim = useRef<Aim | null>(null);
  const hovered = useRef<MenuProperty | null>(null);
  const id = useId();
  const surfaceID = (property: MenuProperty): string => `${id}${property}`;
  // The values open in portals, so the menu finds them by name and not as children.
  const surfaceOf = (property: MenuProperty): Element | null =>
    document.querySelector(`[${SURFACE_ATTRIBUTE}="${surfaceID(property)}"]`);

  useEffect(
    () => () => {
      clearTimeout(restTimer.current);
      clearTimeout(aimTimer.current);
    },
    [],
  );

  const setOpen = (next: MenuProperty | null): void =>
    setState((prev) => transition(prev, next));

  const endAim = (): void => {
    aim.current = null;
    setAiming(false);
  };

  const inPicker = (target: EventTarget): boolean =>
    open != null &&
    target instanceof Element &&
    target.closest(`[${SURFACE_ATTRIBUTE}]`)?.getAttribute(SURFACE_ATTRIBUTE) ===
      surfaceID(open);

  const startAim = (e: PointerEvent<HTMLElement>, from: MenuProperty): void => {
    const picker = surfaceOf(from)?.getBoundingClientRect();
    if (picker == null) return;
    const right = picker.left >= e.clientX;
    const near = right ? picker.left : picker.right;
    const far = right ? picker.right : picker.left;
    aim.current = {
      polygon: [
        { x: e.clientX + (right ? -AIM_BLEED : AIM_BLEED), y: e.clientY },
        { x: near, y: picker.top },
        { x: far, y: picker.top },
        { x: far, y: picker.bottom },
        { x: near, y: picker.bottom },
      ],
      expires: performance.now() + AIM_MS,
      exit: { x: e.clientX, y: e.clientY },
      right,
    };
    setAiming(true);
    clearTimeout(aimTimer.current);
    aimTimer.current = setTimeout(() => {
      if (aim.current == null) return;
      endAim();
      const rest = hovered.current;
      if (rest != null && rest !== from) setOpen(rest);
    }, AIM_MS);
  };

  const leave = (e: PointerEvent<HTMLElement>, property: MenuProperty | null): void => {
    clearTimeout(restTimer.current);
    if (property != null && property === open) startAim(e, property);
  };

  /** @returns whether the pointer still travels toward the open values. */
  const onCourse = (e: PointerEvent<HTMLElement>): boolean => {
    const current = aim.current;
    if (current == null) return false;
    const p = { x: e.clientX, y: e.clientY };
    const travel = current.right ? p.x - current.exit.x : current.exit.x - p.x;
    const live = performance.now() < current.expires && contains(current.polygon, p);
    const still = Math.abs(p.y - current.exit.y) < AIM_MAX_DRIFT;
    return live && (travel >= AIM_MIN_TRAVEL || still);
  };

  const surface: SurfaceHandlers = {
    onClick: (e) => {
      const property = propertyOf(e.target);
      if (property == null) return;
      clearTimeout(restTimer.current);
      setOpen(property);
    },
    onPointerMove: (e) => {
      if (e.pointerType !== "mouse") return;
      if (inPicker(e.target)) {
        clearTimeout(restTimer.current);
        if (aim.current != null) endAim();
        return;
      }
      const property = propertyOf(e.target);
      if (property !== hovered.current) leave(e, hovered.current);
      hovered.current = property;
      if (onCourse(e)) {
        clearTimeout(restTimer.current);
        return;
      }
      if (aim.current != null) endAim();
      clearTimeout(restTimer.current);
      if (property == null || property === open) return;
      restTimer.current = setTimeout(() => setOpen(property), REST_MS);
    },
    onPointerLeave: (e) => {
      if (e.pointerType !== "mouse") return;
      leave(e, hovered.current);
      hovered.current = null;
    },
    onKeyDownCapture: (e) => {
      if (e.key === "ArrowRight") {
        const property = propertyOf(e.target);
        if (property == null) return;
        e.preventDefault();
        clearTimeout(restTimer.current);
        setOpen(property);
        return;
      }
      if (e.key !== "ArrowLeft" || open == null || !inPicker(e.target)) return;
      if (e.target instanceof HTMLInputElement && e.target.value !== "") return;
      e.preventDefault();
      setOpen(null);
      e.currentTarget
        .querySelector<HTMLElement>(`[${PROPERTY_ATTRIBUTE}="${open}"]`)
        ?.focus();
    },
  };

  const frame = (property: MenuProperty) => ({
    visible: open === property,
    onVisibleChange: (next: xstate.SetArg<boolean>) =>
      setState((prev) => {
        const visible = xstate.executeSetter(next, prev.open === property);
        if (visible) return transition(prev, property);
        if (prev.open === property) return transition(prev, null);
        return prev;
      }),
  });

  const picker = (property: MenuProperty) => ({
    surfaceID: surfaceID(property),
    className: CSS.cls(
      CSS.BEM("range-filter", "picker", "beside"),
      swap && open === property && CSS.BEM("range-filter", "picker", "swap"),
    ),
  });

  return { aiming, surface, frame, picker };
};
