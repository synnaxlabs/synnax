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

import {
  Compare,
  toXY,
  XY,
  ZERO_XY,
  TimeStamp,
  TimeSpan,
  Destructor,
} from "@synnaxlabs/x";

import { parseEventKey, Stage, Trigger, TriggerCallback } from "./triggers";

import { useStateRef } from "@/hooks/useStateRef";

type TriggerListen = (callback: TriggerCallback, sequences: Trigger[]) => Destructor;

interface TriggerContextValue {
  listen: TriggerListen;
}

const TriggerContext = createContext<TriggerContextValue | null>(null);

export const useTriggerContext = (): TriggerContextValue => {
  const ctx = useContext(TriggerContext);
  if (ctx == null) throw new Error("TriggerContext not available");
  return ctx;
};

export interface TriggerProviderProps extends PropsWithChildren {
  ref: React.RefObject<HTMLElement>;
}

interface TriggerRefState {
  curr: Trigger;
  prev: Trigger;
  last: TimeStamp;
}

const ZERO_TRIGGER_STATE: TriggerRefState = {
  curr: [],
  prev: [],
  last: new TimeStamp(0),
};

export const TriggersProvider = ({
  children,
}: {
  children: React.ReactNode;
}): JSX.Element => {
  // We track mouse movement to allow for cursor position on keybord events;
  const cursor = useRef<XY>(ZERO_XY);
  const handleMouseMove = useCallback((e: MouseEvent): void => {
    cursor.current = toXY(e);
  }, []);

  // All registered triggers and callbacks
  const registry = useRef<Map<TriggerCallback, Trigger[]>>(new Map());

  // The current trigger.
  const [, setCurr] = useStateRef<TriggerRefState>({ ...ZERO_TRIGGER_STATE });

  const updateListeners = useCallback(
    (trigger: Trigger, stage: Stage, target: HTMLElement): void =>
      registry.current.forEach((triggers, f) => {
        const matches = triggers.filter(
          (t) => Compare.unorderedPrimitiveArrays(t, trigger) === 0
        );
        if (matches.length > 0)
          f({ target, stage, triggers: matches, cursor: cursor.current });
      }),
    []
  );

  const handleKeyDown = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const key = parseEventKey(e);
    setCurr((prev) => {
      const next: Trigger = [...prev.curr, key];
      if (prev.curr.includes(key)) return prev;
      // This is considered a double press.
      if (
        prev.prev.includes(key) &&
        TimeStamp.since(prev.last).valueOf() < TimeSpan.milliseconds(400).valueOf()
      )
        next.push(key);
      const nextState: TriggerRefState = {
        curr: next,
        prev: prev.curr,
        last: new TimeStamp(),
      };
      updateListeners(nextState.curr, "start", e.target as HTMLElement);
      return nextState;
    });
  }, []);

  const handleKeyUp = useCallback(
    (e: KeyboardEvent | MouseEvent): void =>
      setCurr((prev) => {
        updateListeners(prev.curr, "end", e.target as HTMLElement);
        return {
          ...prev,
          curr: [],
          prev: prev.curr,
        };
      }),
    []
  );

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mousedown", handleKeyDown);
    window.addEventListener("mouseup", handleKeyUp);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mousedown", handleKeyDown);
      window.removeEventListener("mouseup", handleKeyUp);
    };
  }, [handleKeyDown, handleKeyUp, handleMouseMove]);

  const listen = useCallback<TriggerListen>((callback, triggers) => {
    registry.current.set(callback, triggers);
    return () => registry.current.delete(callback);
  }, []);

  return (
    <TriggerContext.Provider value={{ listen }}>{children}</TriggerContext.Provider>
  );
};
