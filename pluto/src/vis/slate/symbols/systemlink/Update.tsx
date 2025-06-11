import { Icon } from "@synnaxlabs/media";
import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const configZ = z.object({
  duration: z.number(),
});

export type Config = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<Config>;

export const Update = () => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        backgroundColor: "var(--pluto-secondary-z-20)",
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        height: "8rem",
        padding: "0 1rem",
      }}
      align="center"
      justify="center"
    >
      <Icon.Logo.NI
        style={{
          width: "3.25rem",
          height: "3.25rem",
          color: "var(--pluto-secondary-z)",
        }}
      />
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Update SystemLink Value
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="input" />
    <Handle.Source location="right" id="output" />
  </Align.Pack>
);
