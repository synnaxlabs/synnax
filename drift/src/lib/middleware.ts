import { Middleware, PayloadAction } from '@reduxjs/toolkit';
import { desugarType } from './type';
import { Window } from './window';
import { executeAction } from './slice';

export const middleware = (window: Window): Middleware<unknown, any> => {
  return ({ getState }) =>
    (next) =>
    (action) => {
      let { type: sugaredType } = action;
      const { type, key, fromListener } = desugarType(sugaredType);
      action.type = type;
      // The action is recirculating from our own relay.
      if (key === window.key()) return;
      if (!fromListener) {
        executeAction({ window, action, getState });
        relayAction(window, action);
      }
      return next(action);
    };
};

const relayAction = (window: Window, action: PayloadAction<unknown>) => {
  window.emit({ action, key: window.key() });
};
