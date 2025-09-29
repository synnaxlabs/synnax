import { channel } from "@synnaxlabs/client";
import z from "zod/v4";

import { Handle } from "@/arc/handle";
import { type types } from "@/arc/stage/types";
import { Channel } from "@/channel";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
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
    <Flex.Box pack x align="center" background={1} bordered borderColor={6} rounded={1}>
      <Flex.Box
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
      </Flex.Box>
      <Divider.Divider y color={5} />
      <Flex.Box style={{ padding: "0rem 2rem" }} align="start" empty>
        <Text.Text level="small" weight={500} color={9}>
          Read Channel
        </Text.Text>
        <Text.Text level="h4" weight={450} color={10}>
          {name}
        </Text.Text>
      </Flex.Box>
      <Handle.Sink location="left" id="trigger" />
      <Handle.Source location="right" id="output" />
    </Flex.Box>
  );
};
