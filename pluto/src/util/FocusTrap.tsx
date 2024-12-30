import { FocusTrap as FocusTrapReact, type FocusTrapProps } from "focus-trap-react";
import { type FC } from "react";

import { canDisable } from "@/util/canDisable";

export const FocusTrap = canDisable<FocusTrapProps>(
  FocusTrapReact as unknown as FC<FocusTrapProps>,
);
