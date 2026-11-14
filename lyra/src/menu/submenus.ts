// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, state as xstate, type xy } from "@synnaxlabs/x";
import { type PointerEvent, type RefObject, useEffect, useRef, useState } from "react";

import { context } from "@/context";

/** How long the pointer rests on an item before its submenu opens. */
const REST_MS = 100;
/** How long the pointer may travel toward an open submenu while it stays open. */
const AIM_MS = 300;
/** How far behind the pointer the aim area starts, so a small jitter stays inside. */
const AIM_BLEED = 5;
/** How far the pointer must move toward the submenu to count as travel to it. */
const AIM_MIN_TRAVEL = 2;
/** How far the pointer may drift up or down while it holds still. */
const AIM_MAX_DRIFT = 3;

interface Aim {
  polygon: xy.XY[];
  expires: number;
  /** Where the pointer left the item of the submenu. */
  exit: xy.XY;
  /** Whether the submenu opens to the right of the menu. */
  right: boolean;
}

const inside = (polygon: xy.XY[], p: xy.XY): boolean => {
  let result = false;
  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const a = polygon[i];
    const b = polygon[j];
    if (
      a.y > p.y !== b.y > p.y &&
      p.x < ((b.x - a.x) * (p.y - a.y)) / (b.y - a.y) + a.x
    )
      result = !result;
  }
  return result;
};

interface State {
  open: string | null;
  /** Whether the open submenu replaced another open submenu. */
  swap: boolean;
}

const ZERO_STATE: State = { open: null, swap: false };

const transition = (prev: State, open: string | null): State => {
  if (prev.open === open) return prev;
  return { open, swap: prev.open != null && open != null };
};

/** The pointer handlers that each item of a menu spreads. */
export interface ItemHandlers {
  onPointerMove: (e: PointerEvent<HTMLElement>) => void;
  onPointerLeave: (e: PointerEvent<HTMLElement>) => void;
}

/** The submenus of one menu, of which at most one is open. */
export interface Submenus {
  /** The key of the open submenu. */
  open: string | null;
  /** Whether the open submenu replaced another open submenu. */
  swap: boolean;
  /** Whether the pointer travels toward the open submenu. */
  aiming: boolean;
  /** The key of the item under the pointer. */
  hovered: string | null;
  /** Opens the submenu with the key at once, or closes the open one for null. */
  setOpen: (key: string | null) => void;
  /** Lets the menu find the content of a submenu, which portals out of the menu. */
  register: (
    key: string,
    content: RefObject<HTMLElement | null>,
  ) => destructor.Destructor;
  item: (key: string) => ItemHandlers;
  /** Takes a pointer move inside the content of the open submenu. */
  enter: (e: PointerEvent<HTMLElement>) => void;
  frame: (key: string) => {
    visible: boolean;
    onVisibleChange: xstate.Setter<boolean>;
  };
}

const NOOP_HANDLERS: ItemHandlers = {
  onPointerMove: () => {},
  onPointerLeave: () => {},
};

export const [SubmenusContext, useSubmenusContext] = context.create<Submenus>({
  defaultValue: {
    open: null,
    swap: false,
    aiming: false,
    hovered: null,
    setOpen: () => {},
    register: () => () => {},
    item: () => NOOP_HANDLERS,
    enter: () => {},
    frame: () => ({ visible: false, onVisibleChange: () => {} }),
  },
  displayName: "Menu.SubmenusContext",
});

/** What a submenu gives its trigger and its content. */
export interface SubContextValue {
  itemKey: string;
  trigger: RefObject<HTMLElement | null>;
  content: RefObject<HTMLElement | null>;
  /** Whether the content replaced the content of another submenu. */
  swap: boolean;
}

// The content resets it, so a dialog nested in the content is not the content.
export const [SubContext, useSubContext] = context.create<SubContextValue>({
  displayName: "Menu.SubContext",
  providerName: "Menu.Sub",
});

/**
 * Opens one submenu of a menu at a time. A rest on the item of a submenu opens it, and
 * a rest on any other item closes it. While the pointer moves toward the open submenu,
 * the items it crosses neither light nor open it.
 */
export const useSubmenus = (): Submenus => {
  const [{ open, swap }, setState] = useState<State>(ZERO_STATE);
  const [aiming, setAiming] = useState(false);
  const [hovered, setHovered] = useState<string | null>(null);
  const restTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const aimTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const aim = useRef<Aim | null>(null);
  const hoveredRef = useRef<string | null>(null);
  const contents = useRef(new Map<string, RefObject<HTMLElement | null>>());

  useEffect(
    () => () => {
      clearTimeout(restTimer.current);
      clearTimeout(aimTimer.current);
    },
    [],
  );

  const transitionTo = (next: string | null): void =>
    setState((prev) => transition(prev, next));

  const hover = (key: string | null): void => {
    hoveredRef.current = key;
    setHovered(key);
  };

  const endAim = (): void => {
    aim.current = null;
    setAiming(false);
  };

  // A rest on the item of a submenu opens it, and a rest on any other item closes it.
  const restTarget = (key: string): string | null =>
    contents.current.has(key) ? key : null;

  const startAim = (e: PointerEvent<HTMLElement>, from: string): void => {
    const content = contents.current.get(from)?.current?.getBoundingClientRect();
    if (content == null) return;
    const right = content.left >= e.clientX;
    const near = right ? content.left : content.right;
    const far = right ? content.right : content.left;
    aim.current = {
      polygon: [
        { x: e.clientX + (right ? -AIM_BLEED : AIM_BLEED), y: e.clientY },
        { x: near, y: content.top },
        { x: far, y: content.top },
        { x: far, y: content.bottom },
        { x: near, y: content.bottom },
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
      const rest = hoveredRef.current;
      if (rest != null && rest !== from) transitionTo(restTarget(rest));
    }, AIM_MS);
  };

  const leave = (e: PointerEvent<HTMLElement>, key: string): void => {
    if (hoveredRef.current !== key) return;
    clearTimeout(restTimer.current);
    hover(null);
    if (key === open) startAim(e, key);
  };

  /** @returns whether the pointer still travels toward the open submenu. */
  const onCourse = (e: PointerEvent<HTMLElement>): boolean => {
    const current = aim.current;
    if (current == null) return false;
    const p = { x: e.clientX, y: e.clientY };
    const travel = current.right ? p.x - current.exit.x : current.exit.x - p.x;
    const live = performance.now() < current.expires && inside(current.polygon, p);
    const still = Math.abs(p.y - current.exit.y) < AIM_MAX_DRIFT;
    return live && (travel >= AIM_MIN_TRAVEL || still);
  };

  const move = (e: PointerEvent<HTMLElement>, key: string): void => {
    if (e.pointerType !== "mouse") return;
    const prev = hoveredRef.current;
    if (prev != null && prev !== key) leave(e, prev);
    if (hoveredRef.current !== key) hover(key);
    if (onCourse(e)) {
      clearTimeout(restTimer.current);
      return;
    }
    if (aim.current != null) endAim();
    clearTimeout(restTimer.current);
    const target = restTarget(key);
    if (target === open) return;
    restTimer.current = setTimeout(() => transitionTo(target), REST_MS);
  };

  const setOpen = (key: string | null): void => {
    clearTimeout(restTimer.current);
    if (aim.current != null) endAim();
    transitionTo(key);
  };

  const register = (
    key: string,
    content: RefObject<HTMLElement | null>,
  ): destructor.Destructor => {
    contents.current.set(key, content);
    return () => {
      contents.current.delete(key);
    };
  };

  const item = (key: string): ItemHandlers => ({
    onPointerMove: (e) => move(e, key),
    onPointerLeave: (e) => {
      if (e.pointerType === "mouse") leave(e, key);
    },
  });

  const enter = (e: PointerEvent<HTMLElement>): void => {
    if (e.pointerType !== "mouse") return;
    clearTimeout(restTimer.current);
    if (aim.current != null) endAim();
    if (hoveredRef.current != null) hover(null);
  };

  const frame = (key: string) => ({
    visible: open === key,
    onVisibleChange: (next: xstate.SetArg<boolean>) =>
      setState((prev) => {
        const visible = xstate.executeSetter(next, prev.open === key);
        if (visible) return transition(prev, key);
        if (prev.open === key) return transition(prev, null);
        return prev;
      }),
  });

  return { open, swap, aiming, hovered, setOpen, register, item, enter, frame };
};
