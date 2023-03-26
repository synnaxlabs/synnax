import type {
  PayloadAction,
  Middleware,
  Dispatch,
  ActionCreatorWithPayload,
} from "@reduxjs/toolkit";

export interface MiddlewareEffectArgs<S, P extends any> {
  getState: () => S;
  dispatch: Dispatch<PayloadAction<P>>;
  action: PayloadAction<P>;
}

export type MiddlewareEffect<S, P extends any> = (
  args: MiddlewareEffectArgs<S, P>
) => void;

export const dispatchEffect =
  <S, I, O extends any>(factory: ActionCreatorWithPayload<O>): MiddlewareEffect<S, I> =>
  ({ dispatch, action }) =>
    dispatch(factory(action.payload as unknown as O) as unknown as PayloadAction<I>);

export const effectMiddleware =
  <S, P extends any>(
    deps: string[],
    effects: Array<MiddlewareEffect<S, P>>
  ): Middleware<Record<string, never>, S, Dispatch<PayloadAction<P>>> =>
  ({ getState, dispatch }) =>
  (next) =>
  (action) => {
    const state = next(action);
    if (deps.includes(action.type))
      effects.forEach((factory) => factory({ getState, dispatch, action }));
    return state;
  };
