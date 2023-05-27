import { useContext, createContext, useId, useCallback } from "react";

import { useState } from "@storybook/addons";

import { PseudoInitialState, PseudoSetState } from "@/core/hooks/useStateRef";

export interface VisCanvasContextValue {
  elPath: string;
  keyPath: string;
  supports: (el: string) => boolean;
  set: <T>(key: string, el: string, data: T) => void;
  get: <T>(key: string) => T;
}

export const VisContext = createContext<VisCanvasContextValue | null>(null);

const useVisContext = (): VisCanvasContextValue => {
  const ctx = useContext(VisContext);
  if (ctx == null) throw new Error("useVisContext must be used within a VisProvider");
  return ctx;
};

export interface VisProviderProps {
  path: string;
  keyPath: string;
  supports: (el: string) => boolean;
}

export interface UseVisElementReturn<S> {
  keyPath: string;
  state: [S, PseudoSetState<S>];
}

export const useVisElement = <S extends unknown>(
  el: string,
  initialState: PseudoInitialState<S>
): UseVisElementReturn<S> => {
  const [state, _setState] = useState<S>(initialState);
  const ctx = useVisContext();
  const id = useId();
  validateSupports(ctx, el);
  ctx.set(id, el, state);

  const setState = useCallback(
    (data: S | ((prev: S) => S)): void => {
      setState((prevState) => {
        const nextState = typeof data === "function" ? data(prevState) : data;
        ctx.set(id, el, nextState);
        return nextState;
      });
    },
    [id, ctx, _setState]
  );

  return { keyPath: `${ctx.keyPath}.${id}`, state: [state, setState] };
};

const validateSupports = (ctx: VisCanvasContextValue, el: string): void => {
  if (!ctx.supports(el)) throw new Error(`${el} is not supported by ${ctx.elPath}`);
};

export class WorkerContext {}
