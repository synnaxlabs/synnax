import { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import z from "zod/v4";

import { Align } from "@/align";
import { Channel } from "@/channel";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

export const Symbol = ({ channel }: SymbolProps) => {
  const name = Channel.useName(channel, "Channel");

  return (
    <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
      <Align.Space
        style={{
          backgroundColor: "var(--pluto-primary-z-20)",
          borderTopLeftRadius: "1rem",
          borderBottomLeftRadius: "1rem",
          height: "8rem",
          padding: "0 1rem",
        }}
        align="center"
        justify="center"
      >
        <Icon.Channel
          style={{
            width: "3.25rem",
            height: "3.25rem",
            color: "var(--pluto-primary-z)",
          }}
        />
      </Align.Space>
      <Divider.Divider y shade={5} />
      <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
        <Text.Text level="small" weight={500} shade={9}>
          Read Channel
        </Text.Text>
        <Text.Text level="h4" weight={450} shade={10}>
          {name}
        </Text.Text>
      </Align.Space>
      <Handle.Sink location="left" id="trigger" />
      <Handle.Source location="right" id="value" />
    </Align.Pack>
  );
};
