import "@/caret/Animated.css";

import { Icon } from "@synnaxlabs/media";
import { location } from "@synnaxlabs/x";

import { CSS } from "@/css";

export interface AnimatedProps {
  enabledLoc: location.Location;
  disabledLoc: location.Location;
  enabled: boolean;
}

export const Animated = ({ enabledLoc, disabledLoc, enabled }: AnimatedProps) => (
  <Icon.Caret.Up
    className={CSS(
      CSS.B("caret-animated"),
      CSS.loc(enabled ? enabledLoc : disabledLoc),
    )}
  />
);
