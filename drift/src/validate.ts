import type { AnyAction } from "@reduxjs/toolkit";

const undefinedActionMessage = "[drift] - unexpected undefined action";
const undefinedActionTypeMessage = "[drift] - unexpected undefined action type";

/**
 * Ensures an action is valid, and throws an error if it is not.
 * @param a - The action to validate.
 */
export const validateAction = (meta: {
  emitted?: boolean;
  action?: AnyAction;
  emitter?: string;
}): void => {
  meta.emitted ??= false;
  if (meta.action == null) {
    console.warn(undefinedActionMessage, meta);
    throw new Error(undefinedActionMessage);
  }
  if (meta.action.type == null || meta.action.type.length === 0) {
    console.warn(undefinedActionTypeMessage, meta);
    throw new Error(undefinedActionTypeMessage);
  }
};
