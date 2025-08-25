import { color, TimeSpan } from "@synnaxlabs/x";
import z from "zod/v4";

import { Icon } from "@/icon";
import { Text } from "@/text";
import { Base } from "@/vis/slate/symbols/Base";
import { type types } from "@/vis/slate/symbols/types";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

const BLUE_HEX = color.construct("#0066FF");

export const Interval = ({ duration }: SymbolProps) => (
  <Base
    type="Time Interval"
    Icon={<Icon.Time />}
    color={color.cssString(color.setAlpha(BLUE_HEX, 0.2))}
    textColor={color.cssString(BLUE_HEX)}
    sources={[{ key: "output", Icon: Icon.Time }]}
  >
    <Text.Text level="h4" weight={500} variant="code">
      {new TimeSpan(duration).toString()}
    </Text.Text>
  </Base>
);
