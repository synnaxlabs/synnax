import { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon } from "@synnaxlabs/pluto";
import z from "zod";

import { Align } from "@/align";
import { Channel } from "@/channel";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const config = z.object({
  channels: z.array(channel.keyZ),
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ channels }: SymbolProps) => {
  const name = Channel.useName(channels[0], "Channel");

  return (
    <Align.Pack x align="center" background={1} bordered borderShade={5} rounded={0.5}>
      <PIcon.Icon
        style={{
          padding: "0.5rem",
          paddingBottom: "0.25rem",
          background: "var(--pluto-primary-z)",
          borderTopLeftRadius: "0.5rem",
          borderBottomLeftRadius: "0.5rem",
        }}
      >
        <Icon.Channel
          style={{
            width: "3.25rem",
            height: "3.25rem",
            color: "var(--pluto-gray-l0)",
          }}
        />
      </PIcon.Icon>
      <Align.Space style={{ padding: "0 2rem" }} align="center">
        <Text.Text level="p" weight={450} shade={9} style={{ marginTop: "-0.5rem" }}>
          {name}
        </Text.Text>
      </Align.Space>
      <Handle.Source location="right" id="value" />
    </Align.Pack>
  );
};
