import { direction, location } from "@synnaxlabs/x";
import { z } from "zod";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Grid } from "@/schematic/node/common/grid";
import { telem } from "@/telem/aether";
import { Control } from "@/telem/control";

export const chipConfigZ = z.object({
  source: telem.statusSourceSpecZ.optional(),
  sink: telem.booleanSinkSpecZ.optional(),
});
export type ChipConfig = z.infer<typeof chipConfigZ>;

export const indicatorConfigZ = z.object({
  statusSource: telem.statusSourceSpecZ.optional(),
  colorSource: telem.colorSourceSpecZ.optional(),
});
export type IndicatorConfig = z.infer<typeof indicatorConfigZ>;

export const stateConfigZ = z.object({
  show: z.boolean().optional(),
  showChip: z.boolean().optional(),
  showIndicator: z.boolean().optional(),
  chip: chipConfigZ.optional(),
  indicator: indicatorConfigZ.optional(),
  orientation: location.locationZ.optional(),
});
export type StateConfig = z.infer<typeof stateConfigZ>;

export interface StateProps extends StateConfig, Omit<Flex.BoxProps, "direction"> {
  chip?: Control.ChipProps;
  indicator?: Control.IndicatorProps;
}

export const stateGridItem = (props?: StateConfig): Grid.Item | null => {
  if (props == null) return null;
  const {
    show = true,
    showChip = true,
    showIndicator = true,
    chip,
    indicator,
    orientation = "bottom",
  } = props;
  return {
    key: "control",
    element: (
      <Flex.Box
        direction={direction.swap(orientation)}
        align="center"
        className={CSS(CSS.B("control-state"))}
        gap="small"
      >
        {show && showChip && <Control.Chip size="small" {...chip} />}
        {show && showIndicator && <Control.Indicator {...indicator} />}
      </Flex.Box>
    ),
    location: orientation,
  };
};
