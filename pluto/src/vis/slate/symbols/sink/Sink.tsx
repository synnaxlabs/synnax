import { channel } from "@synnaxlabs/client";
import z from "zod/v4";

import { Channel } from "@/channel";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Base } from "@/vis/slate/symbols/Base";
import { type types } from "@/vis/slate/symbols/types";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ channel }: SymbolProps) => {
  const name = Channel.useName(channel, "Channel");

  return (
    <Base
      type="Sink"
      Icon={<Icon.Control />}
      color="var(--pluto-error-z-20)"
      textColor="var(--pluto-error-z)"
      sinks={[{ key: "value", Icon: Icon.Decimal }]}
    >
      <Text.Text level="p" weight={500} shade={9}>
        {name}
      </Text.Text>
    </Base>
  );
};
