// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  useContext,
  useEffect,
  useCallback,
  useRef,
  PropsWithChildren,
} from "react";

import { toXY, XY, ZERO_XY, TimeStamp, TimeSpan, Destructor } from "@synnaxlabs/x";

import {
  MouseKey,
  MOUSE_KEYS,
  parseEventKey,
  Trigger,
  TriggerCallback,
} from "./triggers";

import { useStateRef } from "@/hooks/useStateRef";

type TriggerListen = (callback: TriggerCallback) => Destructor;

export interface TriggerContextValue {
  listen: TriggerListen;
}

const TriggerContext = createContext<TriggerContextValue | null>(null);

export const useTriggerContext = (): TriggerContextValue => {
  const ctx = useContext(TriggerContext);
  if (ctx == null) throw new Error("TriggerContext not available");
  return ctx;
};

interface TriggerRefState {
  next: Trigger;
  prev: Trigger;
  last: TimeStamp;
}

const ZERO_TRIGGER_STATE: TriggerRefState = {
  next: [],
  prev: [],
  last: new TimeStamp(0),
};

const EXCLUDE_TRIGGERS = ["CapsLock"];

export interface TriggersProviderProps extends PropsWithChildren { }

export const TriggersProvider = ({ children }: TriggersProviderProps): ReactElement => {
  // We track mouse movement to allow for cursor position on keybord events;
  const cursor = useRef<XY>(ZERO_XY);
  const handleMouseMove = useCallback((e: MouseEvent): void => {
    cursor.current = toXY(e);
  }, []);

  // All registered triggers and callbacks
  const registry = useRef<Map<TriggerCallback, null>>(new Map());

  // The current trigger.
  const [, setCurr] = useStateRef<TriggerRefState>({ ...ZERO_TRIGGER_STATE });

  const updateListeners = useCallback(
    (state: TriggerRefState, target: HTMLElement): void => {
      const next = state.next.length > 0 ? [state.next] : [];
      const prev = state.prev.length > 0 ? [state.prev] : [];
      const event = { target, next, prev, cursor: cursor.current };
      registry.current.forEach((_, f) => f(event));
    },
    []
  );

  const handleKeyDown = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const key = parseEventKey(e);
    if (EXCLUDE_TRIGGERS.includes(key as string)) return;
    setCurr((prev) => {
      const next: Trigger = [...prev.next, key];
      if (prev.next.includes(key)) return prev;
      // This is considered a double press.
      if (
        prev.prev.includes(key) &&
        TimeStamp.since(prev.last).valueOf() < TimeSpan.milliseconds(400).valueOf()
      )
        next.push(key);
      const nextState: TriggerRefState = {
        next,
        prev: prev.next,
        last: new TimeStamp(),
      };
      updateListeners(nextState, e.target as HTMLElement);
      return nextState;
    });
  }, []);

  const handleKeyUp = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const key = parseEventKey(e);
    if (EXCLUDE_TRIGGERS.includes(key as string)) return;
    setCurr((prevS) => {
      const next = prevS.next.filter(
        (k) => k !== key && !MOUSE_KEYS.includes(k as MouseKey)
      );
      const prev = prevS.next;
      const nextS: TriggerRefState = {
        ...prevS,
        next,
        prev,
      };
      updateListeners(nextS, e.target as HTMLElement);
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
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mousedown", handleKeyDown);
      window.removeEventListener("mouseup", handleKeyUp);
      window.removeEventListener("dragend", handleKeyUp);
      window.removeEventListener("drop", handleKeyUp);
    };
  }, [handleKeyDown, handleKeyUp, handleMouseMove]);

  const listen = useCallback<TriggerListen>((callback) => {
    registry.current.set(callback, null);
    return () => registry.current.delete(callback);
  }, []);

  return (
    <TriggerContext.Provider value={{ listen }}>{children}</TriggerContext.Provider>
  );
};
