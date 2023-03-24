import { Middleware, Dispatch, AnyAction } from "@reduxjs/toolkit";

export const layoutRemoverEffectMiddleware =
  
  factory: 
): Middleware<
    DispatchExt,
    S,
    D
  > =>
  (store) =>
  (next) =>
  (action) => {
    const state = next(action);
    if (action.type === "layout/removeEffect") {
      const { effect } = action.payload;
      const state = store.getState();
      const { effects } = state.layout;
      const newEffects = effects.filter((e) => e !== effect);
      store.dispatch(layoutSlice.actions.setEffects(newEffects));
    }
    return state;
  };
