import { color } from "@synnaxlabs/x";
import z from "zod";

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

const PURPLE_HEX = color.construct("#635BFF");

export const Count = () => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        height: "8rem",
        backgroundColor: color.cssString(color.setAlpha(PURPLE_HEX, 0.2)),
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
      }}
      align="center"
      justify="center"
    >
      <Text.Text
        level="h4"
        style={{
          color: color.cssString(PURPLE_HEX),
          borderTopLeftRadius: "1rem",
          borderBottomLeftRadius: "1rem",
          padding: "0.25rem 1rem",
        }}
      >
        C
      </Text.Text>
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Count
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="input" />
    <Handle.Source location="right" id="output" />
  </Align.Pack>
);
