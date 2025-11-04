// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor, TimeSpan, TimeStamp, xy } from "@synnaxlabs/x";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";

import { useStateRef } from "@/hooks/ref";
import {
  type Callback,
  eventKey,
  isAlphanumericKey,
  type Key,
  match,
  type MatchOptions,
  MOUSE_KEYS,
  type MouseKey,
  type Trigger,
} from "@/triggers/triggers";

export interface Listen {
  (callback: Callback): Destructor;
}

export interface ContextValue {
  listen: Listen;
}

const ZERO_CONTEXT_VALUE: ContextValue = {
  listen: () => () => {},
};

const Context = createContext<ContextValue>(ZERO_CONTEXT_VALUE);

export const useContext = () => use(Context);

interface RefState {
  next: Trigger;
  prev: Trigger;
  last: TimeStamp;
}

const ZERO_REF_STATE: RefState = {
  next: [],
  prev: [],
  last: new TimeStamp(0),
};

const EXCLUDE_TRIGGERS = ["CapsLock"];

export interface ProviderProps extends PropsWithChildren {
  preventDefaultOn?: Trigger[];
  preventDefaultOptions?: MatchOptions;
}

const isInputOrContentEditable = (e: KeyboardEvent): boolean => {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement)
    return true;
  const isHTMLElement = e.target instanceof HTMLElement;
  if (
    isHTMLElement &&
    (e.target.getAttribute("contenteditable") === "true" || e.target.role === "textbox")
  )
    return true;
  return false;
};

const shouldTriggerOnKeyDown = (key: Key, e: KeyboardEvent): boolean => {
  if (EXCLUDE_TRIGGERS.includes(key)) return false;
  if (isInputOrContentEditable(e)) {
    // If there is an alphanumeric key and the user is not holding down ctrl or meta,
    // we don't want to trigger the key.
    if (isAlphanumericKey(key) && !e.ctrlKey && !e.metaKey) return false;
    return true;
  }
  return true;
};

export const Provider = ({
  children,
  preventDefaultOn,
  preventDefaultOptions,
}: ProviderProps): ReactElement => {
  // We track mouse movement to allow for cursor position on keyboard events;
  const cursor = useRef<xy.XY>(xy.ZERO);
  const handleMouseMove = useCallback((e: MouseEvent): void => {
    cursor.current = xy.construct(e);
  }, []);

  // All registered triggers and callbacks
  const registry = useRef<Map<Callback, null>>(new Map());

  // The current trigger.
  const [, setCurr] = useStateRef<RefState>({ ...ZERO_REF_STATE });

  const updateListeners = useCallback((state: RefState, target: HTMLElement): void => {
    const next = state.next.length > 0 ? [state.next] : [];
    const prev = state.prev.length > 0 ? [state.prev] : [];
    const event = { target, next, prev, cursor: cursor.current };
    registry.current.forEach((_, f) => f(event));
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const key = eventKey(e);
    // We prevent the default behavior of arrow keys to prevent scrolling and movement
    // of the cursor. We might want to move this elsewhere in the future.
    if (["ArrowUp", "ArrowDown"].includes(key)) e.preventDefault();
    // We don't want to trigger any events for excluded keys.
    // If our target element is an input, we don't want to trigger any events.
    if (!shouldTriggerOnKeyDown(key, e as KeyboardEvent)) return;
    setCurr((prev) => {
      const next: Trigger = [...prev.next, key];
      if (prev.next.includes(key)) return prev;
      // This is considered a double press.
      if (
        prev.prev.includes(key) &&
        TimeStamp.since(prev.last).valueOf() < TimeSpan.milliseconds(300).valueOf()
      )
        next.push(key);
      const nextState: RefState = {
        next,
        prev: prev.next,
        last: new TimeStamp(),
      };
      if (shouldPreventDefault(next, preventDefaultOn, preventDefaultOptions))
        e.preventDefault();
      updateListeners(nextState, e.target as HTMLElement);
      return nextState;
    });
  }, []);

  const handleKeyUp = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const key = eventKey(e);
    // We prevent the default behavior of arrow keys to prevent scrolling and movement
    if (["ArrowUp", "ArrowDown"].includes(key)) e.preventDefault();
    // We don't want to trigger any events for excluded keys.
    if (EXCLUDE_TRIGGERS.includes(key)) return;
    setCurr((prevS) => {
      let next = prevS.next.filter(
        (k) => k !== key && !MOUSE_KEYS.includes(k as MouseKey),
      );
      // Later versions of Safari have a 'sticky shift' phenomenon when the Shift key
      // key up even it not always fired. To correct for this, we manually check for
      // the event.shiftKey flag.
      if (!e.shiftKey && next.includes("Shift"))
        next = next.filter((k) => k !== "Shift");
      const prev = prevS.next;
      const nextS: RefState = { ...prevS, next, prev };
      if (shouldPreventDefault(next, preventDefaultOn, preventDefaultOptions))
        e.preventDefault();
      updateListeners(nextS, e.target as HTMLElement);
      return nextS;
    });
  }, []);

  /**
   * If the mouse leaves the window, we want to clear all triggers. This prevents
   * issues with the user holding down a key and then moving the mouse out of the
   * window.
   */
  const handlePageVisibility = useCallback((event: Event): void => {
    setCurr((prevS) => {
      const prev = prevS.next;
      const nextS: RefState = { ...prevS, next: [], prev };
      updateListeners(nextS, event.target as HTMLElement);
      return nextS;
    });
  }, []);

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mousedown", handleKeyDown);
    window.addEventListener("mouseup", handleKeyUp);
    window.addEventListener("dragend", handleKeyUp);
    window.addEventListener("drop", handleKeyUp);
    window.addEventListener("blur", handlePageVisibility);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mousedown", handleKeyDown);
      window.removeEventListener("mouseup", handleKeyUp);
      window.removeEventListener("dragend", handleKeyUp);
      window.removeEventListener("drop", handleKeyUp);
      window.removeEventListener("blur", handlePageVisibility);
    };
  }, [handleKeyDown, handleKeyUp, handleMouseMove]);

  const listen = useCallback<Listen>((callback) => {
    registry.current.set(callback, null);
    return () => registry.current.delete(callback);
  }, []);

  const ctxValue = useMemo(() => ({ listen }), [listen]);

  return <Context value={ctxValue}>{children}</Context>;
};

const shouldPreventDefault = (
  t: Trigger,
  preventDefaultOn?: Trigger[],
  preventDefaultOptions?: MatchOptions,
): boolean =>
  preventDefaultOn != null && match([t], preventDefaultOn, preventDefaultOptions);
