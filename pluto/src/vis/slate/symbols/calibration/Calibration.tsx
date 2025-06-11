import { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export const Symbol = () => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
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
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Query Calibrations
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="value" />
    <Handle.Source location="right" id="output" />
  </Align.Pack>
);
