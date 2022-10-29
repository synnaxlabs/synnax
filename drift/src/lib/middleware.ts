import { Action, AnyAction, Dispatch, Middleware } from '@reduxjs/toolkit';

import {
  executeAction,
  isDriftAction,
  maybeSetWindowKey,
  setWindowKey,
  StoreState,
} from './slice';
import { desugarType } from './type';
import { Window } from './window';

export const middleware = <S extends StoreState, A extends Action = AnyAction>(
  window: Window<S, A>
  // eslint-disable-next-line @typescript-eslint/ban-types
): Middleware<{}, S, Dispatch<A>> => {
  return ({ getState }) =>
    (next) =>
    (action) => {
      const { type: sugaredType } = action;
      const { type, key, fromListener } = desugarType(sugaredType);
      action.type = type;

      // The action is recirculating from our own relay.
      if (key === window.key()) return;

      if (isDriftAction(action.type)) {
        if (!fromListener) {
          action = maybeSetWindowKey(window, action, getState);
        }
        if (window.isMain()) {
          executeAction({
            window,
            action,
            getState,
          });
        }
      }

      const res = next(action);

      if (!fromListener && action.type !== setWindowKey.type)
        window.emit({ action, key: window.key() });

      return res;
    };
};
