import { ReactElement, memo } from "react";

import { ColorT } from "@/core/color";
import { Pack, PackProps, Typography } from "@/core/std";
import { Theming } from "@/core/theming";
import { Virtual } from "@/core/virtual/main";
import {
  Value as WorkerValue,
  ValueState as WorkerValueState,
} from "@/core/vis/pid/Value/worker";
import { ComponentSize } from "@/util/component";

export interface ValueProps
  extends Omit<WorkerValueState, "font" | "color">,
    Omit<PackProps, "color"> {
  color?: ColorT;
  size?: ComponentSize;
}

export const Value = memo(
  ({ label, color, size = "medium", ...props }: ValueProps): ReactElement => {
    const theme = Theming.use();
    Virtual.useComponent(WorkerValue.TYPE, {
      font: Typography.themeFont(theme, Typography.ComponentSizeLevels[size]),
      color: color ?? theme.colors.text,
      label,
      ...props,
    });

    return (
      <Pack {...props} size={size} direction="y">
        {label != null && (
          <Typography.Text
            level="p"
            color={color}
            style={{ padding: "0 1rem", textAlign: "center" }}
          >
            {label}
          </Typography.Text>
        )}
        <div></div>
      </Pack>
    );
  }
);
Value.displayName = "Value";
