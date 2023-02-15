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
  useState,
  RefObject,
  MutableRefObject,
} from "react";

import { comparePrimitiveArrays } from "@synnaxlabs/x";

import { Key, Modifier, Stage, Trigger } from "./types";

import { toXY, XY, ZERO_XY } from "@/spatial";

export interface TriggerEvent {
  target: HTMLElement;
  triggers: Trigger[];
  stage: Stage;
  cursor: XY;
}

export type TriggerCallback = (e: TriggerEvent) => void;

type Destructor = () => void;
type TriggerListen = (callback: TriggerCallback, sequences: Trigger[]) => Destructor;

interface TriggerContextValue {
  listen: TriggerListen;
}

const TriggerContext = createContext<TriggerContextValue | null>(null);

const useTriggerContext = (): TriggerContextValue => {
  const ctx = useContext(TriggerContext);
  if (ctx == null) throw new Error("TriggerContext not available");
  return ctx;
};

export const useTrigger = (
  triggers: Trigger[],
  callback?: TriggerCallback,
  bound?: RefObject<HTMLElement>
): MutableRefObject<UseTriggerHeldReturn> => {
  const { listen } = useTriggerContext();
  const ref = useRef<UseTriggerHeldReturn>({ triggers: [], held: false });

  useEffect(
    () =>
      listen((e) => {
        if (bound != null) {
          if (bound.current == null) return;
          if (
            (e.stage === "start" || !ref.current.held) &&
            !bound.current.contains(e.target) &&
            e.target !== bound.current
          )
            return;
        }
        ref.current = {
          triggers: e.stage === "end" ? [] : triggers,
          held: e.stage === "start",
        };
        callback?.(e);
      }, triggers),
    [callback, triggers, listen]
  );
  return ref;
};

export interface UseTriggerHeldReturn {
  triggers: Trigger[];
  held: boolean;
}

export const useTriggerHeld = (triggers: Trigger[]): UseTriggerHeldReturn => {
  const [held, setHeld] = useState<UseTriggerHeldReturn>({ triggers: [], held: false });
  useTrigger(triggers, ({ triggers, stage }) =>
    setHeld({ triggers: stage === "end" ? [] : triggers, held: stage === "start" })
  );
  return held;
};

export interface TriggerProviderProps extends PropsWithChildren {
  ref: React.RefObject<HTMLElement>;
}

export const TriggersProvider = ({
  children,
}: {
  children: React.ReactNode;
}): JSX.Element => {
  const registry = useRef<Map<TriggerCallback, Trigger[]>>(new Map());
  const cursor = useRef<XY>(ZERO_XY);
  const curr = useRef<Trigger | null>();

  const handleMouseMove = useCallback((e: MouseEvent): void => {
    cursor.current = toXY(e);
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const trigger = parseEvent(e);
    if (curr.current != null && comparePrimitiveArrays(curr.current, trigger) === 0)
      return;
    curr.current = trigger;
    registry.current.forEach((triggers, f) => {
      const matches = triggers.filter((t) => comparePrimitiveArrays(t, trigger) === 0);
      if (matches.length > 0)
        f({
          target: e.target as HTMLElement,
          triggers: matches,
          stage: "start",
          cursor: cursor.current,
        });
    });
  }, []);

  const handleKeyUp = useCallback((e: KeyboardEvent | MouseEvent): void => {
    const trigger = parseEvent(e);
    const checkTrigger = curr.current ?? trigger;
    registry.current.forEach((triggers, f) => {
      const matches = triggers.filter(
        (t) => comparePrimitiveArrays(t, checkTrigger) === 0
      );
      if (matches.length > 0)
        f({
          target: e.target as HTMLElement,
          triggers: matches,
          stage: "end",
          cursor: cursor.current,
        });
    });
    if (curr.current == null) return;
    curr.current =
      comparePrimitiveArrays(curr.current, trigger) === 0 ? null : curr.current;
  }, []);

  const handleDoubleClick = useCallback((e: MouseEvent): void => {
    const trigger = parseEvent(e);
    trigger[0] = "MouseDouble";
    curr.current = null;
    registry.current.forEach((triggers, f) => {
      const matches = triggers.filter(
        (t) => comparePrimitiveArrays(t, trigger) === 0 || t[1] === trigger[0]
      );
      if (matches.length > 0)
        f({
          target: e.target as HTMLElement,
          triggers: matches,
          stage: "end",
          cursor: cursor.current,
        });
    });
  }, []);

  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("keyup", handleKeyUp);
    window.addEventListener("mousedown", handleKeyDown);
    window.addEventListener("mouseup", handleKeyUp);
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("dblclick", handleDoubleClick);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("keyup", handleKeyUp);
      window.removeEventListener("mousedown", handleKeyDown);
      window.removeEventListener("mouseup", handleKeyUp);
      window.removeEventListener("mousemove", handleMouseMove);
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

const parseEvent = (e: KeyboardEvent | MouseEvent): Trigger => {
  const modifier = parseModifier(e);
  const key =
    e instanceof KeyboardEvent ? parseKeyBoardEventKey(e) : parseMouseEventKey(e);
  if (key === modifier) return [key, null];
  return [key, modifier];
};

const parseMouseEventKey = (e: MouseEvent): Key => {
  if (e.button === 0) return "MouseLeft";
  if (e.button === 1) return "MouseMiddle";
  if (e.button === 2) return "MouseRight";
  throw new Error(`Invalid mouse button: ${e.button}`);
};

const parseKeyBoardEventKey = (e: KeyboardEvent): Key =>
  (e.key[0].toUpperCase() + e.key.slice(1)) as Key;

const parseModifier = (e: KeyboardEvent | MouseEvent): Modifier =>
  e.shiftKey ? "Shift" : e.ctrlKey ? "Control" : e.altKey ? "Alt" : null;

export const matchTriggers = (triggers: Trigger[], candidates: Trigger[]): boolean =>
  triggers.some((t) => candidates.some((c) => comparePrimitiveArrays(t, c) === 0));
