import { color, status } from "@synnaxlabs/x";
import z from "zod/v4";

import { Align } from "@/align";
import { Icon } from "@/icon";
import { Status } from "@/status";
import { Text } from "@/text";
import { Base } from "@/vis/slate/symbols/Base";
import { type types } from "@/vis/slate/symbols/types";

export const config = z.object({
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
    sinks={[{ key: "value", Icon: Icon.Number }]}
  >
    <Align.Space x align="center" justify="center" gap="tiny">
      <Status.Indicator variant={variant} size="2.5em" />
      <Text.Text level="p" weight={500} shade={11}>
        {message}
      </Text.Text>
    </Align.Space>
  </Base>
);
