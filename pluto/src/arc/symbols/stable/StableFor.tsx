import { color, TimeSpan } from "@synnaxlabs/x";
import z from "zod/v4";

import { Base } from "@/arc/symbols/Base";
import { type types } from "@/arc/symbols/types";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

const PURPLE_HEX = color.construct("#635BFF");

export const StableFor = ({ duration }: SymbolProps) => (
  <Base
    type="Stable For"
    Icon={<Icon.Time />}
    color={color.cssString(color.setAlpha(PURPLE_HEX, 0.2))}
    textColor={color.cssString(PURPLE_HEX)}
    sinks={[{ key: "input", Icon: Icon.Number }]}
    sources={[{ key: "value", Icon: Icon.Number }]}
  >
    <Text.Text level="h4" weight={500} variant="code">
      {new TimeSpan(duration).toString()}
    </Text.Text>
  </Base>
);
