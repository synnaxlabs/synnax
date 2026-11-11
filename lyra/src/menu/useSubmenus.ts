// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { state as xstate, type xy } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type PointerEvent,
  type RefCallback,
  useEffect,
  useRef,
  useState,
} from "react";

import { context } from "@/context";

/** How long the pointer rests on an item before it opens or closes a submenu. */
const REST_MS = 100;
/** How long the pointer may travel toward an open submenu while it stays open. */
const AIM_MS = 300;
/** How far behind the pointer the aim area starts, so a small jitter stays inside. */
const AIM_BLEED = 5;
/** How far the pointer must move toward the submenu to count as travel to it. */
const AIM_MIN_TRAVEL = 2;
/** How far the pointer may drift up or down while it holds still. */
const AIM_MAX_DRIFT = 3;

/** Marks every item while the pointer travels toward an open submenu. */
const AIMING_ATTRIBUTE = "data-aiming";

interface Aim {
  polygon: xy.XY[];
  expires: number;
  /** Where the pointer left the item. */
  exit: xy.XY;
  /** Whether the submenu opens to the right of the menu. */
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

interface State {
  open: string | null;
  /** Whether the open submenu replaced another one. */
  swap: boolean;
}

const INITIAL_STATE: State = { open: null, swap: false };

const transition = (prev: State, open: string | null): State => {
  if (prev.open === open) return prev;
  return { open, swap: prev.open != null && open != null };
};

/** The handlers a menu item spreads, so a rest on it closes an open submenu. */
export interface ItemHandlers {
  ref: RefCallback<HTMLElement>;
  onPointerEnter: () => void;
  onPointerMove: (e: PointerEvent<HTMLElement>) => void;
  onPointerLeave: (e: PointerEvent<HTMLElement>) => void;
}

/** The handlers an item spreads to open its submenu. */
export interface TriggerHandlers extends ItemHandlers {
  onClick: () => void;
  onKeyDown: (e: KeyboardEvent<HTMLElement>) => void;
}

/** Opens the submenus of one menu, one at a time. */
export interface Controller extends State {
  setVisible: (key: string, next: xstate.SetArg<boolean>) => void;
  item: () => ItemHandlers;
  trigger: (key: string) => TriggerHandlers;
  /** Holds the surface of the submenu with the key, which the pointer aims at. */
  surfaceRef: (key: string) => RefCallback<HTMLElement>;
}

export const [ControllerContext, useControllerContext] = context.create<Controller>({
  displayName: "Menu.SubmenusContext",
  providerName: "Menu.Menu",
});

/** The item under the pointer. A plain item opens no submenu, so its key is null. */
interface Hover {
  key: string | null;
}

/**
 * Opens one submenu at a time. A rest on an item opens its submenu, or closes the open
 * one if the item has none, and a path toward the open submenu keeps it open. The aim
 * lives outside React state, so the pointer re-renders nothing while it moves.
 */
export const useSubmenus = (): Controller => {
  const [{ open, swap }, setState] = useState<State>(INITIAL_STATE);
  const restTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const aimTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const aim = useRef<Aim | null>(null);
  const items = useRef(new Set<HTMLElement>());
  const triggers = useRef(new Map<string, HTMLElement>());
  const surfaces = useRef(new Map<string, HTMLElement>());
  const hovered = useRef<Hover | null>(null);

  const endAim = (): void => {
    aim.current = null;
    items.current.forEach((el) => el.removeAttribute(AIMING_ATTRIBUTE));
  };

  useEffect(
    () => () => {
      clearTimeout(restTimer.current);
      clearTimeout(aimTimer.current);
    },
    [],
  );

  useEffect(() => {
    if (open == null) return;
    const inSurface = (target: EventTarget | null): boolean =>
      target instanceof Node && surfaces.current.get(open)?.contains(target) === true;
    const handlePointerMove = (e: globalThis.PointerEvent): void => {
      if (!inSurface(e.target)) return;
      clearTimeout(restTimer.current);
      endAim();
    };
    const handleKeyDown = (e: globalThis.KeyboardEvent): void => {
      if (e.key !== "ArrowLeft") return;
      const trigger = triggers.current.get(open);
      if (e.target !== trigger && !inSurface(e.target)) return;
      if (e.target instanceof HTMLInputElement && e.target.value !== "") return;
      e.preventDefault();
      setState((prev) => transition(prev, null));
      trigger?.focus();
    };
    document.addEventListener("pointermove", handlePointerMove);
    document.addEventListener("keydown", handleKeyDown, true);
    return () => {
      document.removeEventListener("pointermove", handlePointerMove);
      document.removeEventListener("keydown", handleKeyDown, true);
    };
  }, [open]);

  const setOpen = (key: string | null): void =>
    setState((prev) => transition(prev, key));

  const startAim = (next: Aim, key: string): void => {
    aim.current = next;
    items.current.forEach((el) => el.setAttribute(AIMING_ATTRIBUTE, ""));
    clearTimeout(aimTimer.current);
    aimTimer.current = setTimeout(() => {
      if (aim.current == null) return;
      endAim();
      const rest = hovered.current;
      if (rest != null && rest.key !== key) setOpen(rest.key);
    }, AIM_MS);
  };

  const setVisible = (key: string, next: xstate.SetArg<boolean>): void =>
    setState((prev) => {
      const visible = xstate.executeSetter(next, prev.open === key);
      if (visible) return transition(prev, key);
      if (prev.open === key) return transition(prev, null);
      return prev;
    });

  const hover = (key: string | null): ItemHandlers => ({
    ref: (el) => {
      if (el == null) return;
      items.current.add(el);
      if (key != null) triggers.current.set(key, el);
      return () => {
        items.current.delete(el);
        if (key != null && triggers.current.get(key) === el)
          triggers.current.delete(key);
      };
    },
    onPointerEnter: () => {
      hovered.current = { key };
    },
    onPointerMove: (e) => {
      if (e.pointerType !== "mouse") return;
      const current = aim.current;
      if (current != null) {
        const p = { x: e.clientX, y: e.clientY };
        const travel = current.right ? p.x - current.exit.x : current.exit.x - p.x;
        const live =
          performance.now() < current.expires && contains(current.polygon, p);
        const still = Math.abs(p.y - current.exit.y) < AIM_MAX_DRIFT;
        if (live && (travel >= AIM_MIN_TRAVEL || still)) {
          clearTimeout(restTimer.current);
          return;
        }
        endAim();
      }
      clearTimeout(restTimer.current);
      if (open === key) return;
      restTimer.current = setTimeout(() => setOpen(key), REST_MS);
    },
    onPointerLeave: (e) => {
      clearTimeout(restTimer.current);
      if (hovered.current?.key === key) hovered.current = null;
      if (key == null || open !== key || e.pointerType !== "mouse") return;
      const surface = surfaces.current.get(key)?.getBoundingClientRect();
      if (surface == null) return;
      const right = surface.left >= e.clientX;
      const near = right ? surface.left : surface.right;
      const far = right ? surface.right : surface.left;
      startAim(
        {
          polygon: [
            { x: e.clientX + (right ? -AIM_BLEED : AIM_BLEED), y: e.clientY },
            { x: near, y: surface.top },
            { x: far, y: surface.top },
            { x: far, y: surface.bottom },
            { x: near, y: surface.bottom },
          ],
          expires: performance.now() + AIM_MS,
          exit: { x: e.clientX, y: e.clientY },
          right,
        },
        key,
      );
    },
  });

  const trigger = (key: string): TriggerHandlers => ({
    ...hover(key),
    onClick: () => {
      clearTimeout(restTimer.current);
      setOpen(key);
    },
    onKeyDown: (e) => {
      if (e.key !== "ArrowRight") return;
      e.preventDefault();
      clearTimeout(restTimer.current);
      setOpen(key);
    },
  });

  const surfaceRef =
    (key: string): RefCallback<HTMLElement> =>
    (el) => {
      if (el == null) surfaces.current.delete(key);
      else surfaces.current.set(key, el);
    };

  return { open, swap, setVisible, item: () => hover(null), trigger, surfaceRef };
};
