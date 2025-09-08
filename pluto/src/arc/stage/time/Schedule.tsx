import { color, TimeSpan } from "@synnaxlabs/x";
import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const configZ = z.object({
  interval: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

const GREEN_HEX = color.construct("#00BFA5");

export const Schedule = ({ interval }: SymbolProps) => (
  <Base
    type="Schedule"
    Icon={<Icon.Time />}
    color={color.cssString(color.setAlpha(GREEN_HEX, 0.2))}
    textColor={color.cssString(GREEN_HEX)}
    sources={[{ key: "output", Icon: Icon.Time }]}
  >
    <Text.Text level="h4" weight={500} variant="code">
      {new TimeSpan(interval).toString()}
    </Text.Text>
  </Base>
);
