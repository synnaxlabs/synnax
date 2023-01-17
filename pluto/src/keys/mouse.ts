import { KeyboardKey } from "./types";

export const MOUSE_WHEEL_CLICK = "Wheel";
export const MOUSE_LEFT_CLCIK = "Left";
export const MOUSE_RIGHT_CLICK = "Right";

export const mouseDownToKey = (e: { button: number }): KeyboardKey => {
  switch (e.button) {
    case 0:
      return MOUSE_LEFT_CLCIK;
    case 1:
      return MOUSE_WHEEL_CLICK;
    case 2:
      return MOUSE_RIGHT_CLICK;
    default:
      return "";
  }
};
