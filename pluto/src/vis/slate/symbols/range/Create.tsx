import { ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { color } from "@synnaxlabs/x";
import z from "zod";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";
import { type types } from "@/vis/slate/symbols/types";

export const configZ = z.object({
  range: ranger.payloadZ.partial(),
});

export type CreateConfig = z.infer<typeof configZ>;

export type SymbolProps = types.SymbolProps<CreateConfig> & {
  start?: number;
  end?: number;
};

const ORANGE_HEX = color.construct("#FF6B00");

export interface CreateProps extends types.SymbolProps<CreateConfig> {}

export const Create = () => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        height: "8rem",
        width: "4rem",
        backgroundColor: color.cssString(color.setAlpha(ORANGE_HEX, 0.2)),
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        position: "relative",
      }}
      align="center"
      justify="center"
    >
      <Icon.Range
        style={{
          width: "2.5rem",
          height: "2.5rem",
          color: color.cssString(ORANGE_HEX),
        }}
      />
      <Handle.Sink
        location="left"
        id="start"
        style={{ position: "absolute", top: "30%" }}
      />
      <Handle.Sink
        location="left"
        id="end"
        style={{ position: "absolute", top: "70%" }}
      />
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Range
      </Text.Text>
    </Align.Space>
    <Handle.Source location="right" id="output" />
  </Align.Pack>
);
