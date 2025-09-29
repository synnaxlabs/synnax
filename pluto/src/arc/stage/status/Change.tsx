import { color, status } from "@synnaxlabs/x";
import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Icon } from "@/icon";
import { Status } from "@/status";
import { Text } from "@/text";

export const config = z.object({
  statusKey: z.string(),
  variant: status.variantZ,
  message: z.string(),
  description: z.string().optional(),
});

export interface Config extends z.infer<typeof config> {}

const RED_HEX = color.construct("#DC136C");

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ message, variant }: SymbolProps) => (
  <Base
    type="Change Status"
    Icon={<Icon.Notification />}
    color={color.cssString(color.setAlpha(RED_HEX, 0.2))}
    textColor={color.cssString(RED_HEX)}
    sinks={[{ key: "output", Icon: Icon.Value }]}
  >
    <Text.Text level="p" weight={500} color={11}>
      <Status.Indicator variant={variant} size="2.5em" />
      {message}
    </Text.Text>
  </Base>
);
