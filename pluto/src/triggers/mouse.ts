import { Key } from "./types";

export const mouseButtonKey = (e: { button: number }): Key => {
  switch (e.button) {
    case 1:
      return "MouseMiddle";
    case 2:
      return "MouseRight";
    default:
      return "MouseLeft";
  }
};
