import { channel } from "@synnaxlabs/client";
import z from "zod/v4";

import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export const Symbol = () => (
  <Flex.Box x pack align="center" background={1} bordered borderColor={6} rounded={1}>
    <Flex.Box
      style={{
        backgroundColor: "var(--pluto-primary-l8-20)",
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        height: "8rem",
        padding: "0 1rem",
      }}
      align="center"
      justify="center"
    >
      <Icon.Cluster
        style={{
          width: "3.25rem",
          height: "3.25rem",
          color: "var(--pluto-gray-l8)",
        }}
      />
    </Flex.Box>
    <Divider.Divider y color={5} />
    <Flex.Box style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} color={9}>
        Query Calibrations
      </Text.Text>
    </Flex.Box>
    <Handle.Sink location="left" id="value" />
    <Handle.Source location="right" id="output" />
  </Flex.Box>
);
