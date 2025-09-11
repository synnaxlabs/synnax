import { channel } from "@synnaxlabs/client";
import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Channel } from "@/channel";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ channel }: SymbolProps) => {
  const name = Channel.useRetrieve({ key: channel }).data?.name ?? "Channel";

  return (
    <Base
      type="Source"
      Icon={<Icon.Channel />}
      color="var(--pluto-primary-z-20)"
      textColor="var(--pluto-primary-z)"
      sources={[{ key: "value", Icon: Icon.Number }]}
    >
      <Text.Text level="p" weight={500} color={10}>
        {name}
      </Text.Text>
    </Base>
  );
};
