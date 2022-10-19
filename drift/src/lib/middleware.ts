import { Middleware, PayloadAction } from '@reduxjs/toolkit';
import { parseDriftMD } from './actions';
import { Runtime } from './runtime';
import { executeAction, isDriftAction } from './slice';

export const middleware = ({
  runtime,
}: {
  runtime: Runtime;
}): Middleware<unknown, any> => {
  return ({ getState }) =>
    (next) =>
    (action) => {
      let { type } = action;
      const { baseType, winID, fromListener } = parseDriftMD(type);
      const isDrift = isDriftAction(baseType);
      action.type = baseType;
      if (winID === runtime.winKey()) return;
      if (!fromListener) {
        if (isDrift) executeAction({ runtime, action, getState });
        relayAction(runtime, action);
      }
      return next(action);
    };
};

const relayAction = (runtime: Runtime, action: PayloadAction<unknown>) => {
  runtime.emit({ action, winKey: runtime.winKey() });
};
