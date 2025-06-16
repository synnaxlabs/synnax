import { ranger } from "@synnaxlabs/client";
import z from "zod/v4";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Icon } from "@/icon";
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

export interface CreateProps extends types.SymbolProps<CreateConfig> {}

export const Create = ({ range }: CreateProps) => (
  <Align.Pack x align="center" background={1} bordered borderShade={6} rounded={1}>
    <Align.Space
      style={{
        height: "12.5rem",
        width: "4rem",
        padding: "0.75rem 0",
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        position: "relative",
        backgroundColor: "var(--pluto-gray-l2)",
      }}
      align="center"
      justify="spaceBetween"
      y
      size={0.5}
    >
      <Icon.ToDo size="2.5em" />
      <Divider.Divider x shade={5} />
      <Icon.InProgress size="2.5em" />
      <Divider.Divider x shade={5} />
      <Icon.Completed size="2.5em" />
      <Handle.Sink
        location="left"
        id="to_do"
        style={{ position: "absolute", top: "16%" }}
      />
      <Handle.Sink
        location="left"
        id="in_progress"
        style={{ position: "absolute", top: "50%" }}
      />
      <Handle.Sink
        location="left"
        id="completed"
        style={{ position: "absolute", top: "85%" }}
      />
    </Align.Space>
    <Divider.Divider y shade={8} />
    <Align.Space style={{ padding: "0rem 2rem" }} align="start" empty>
      <Text.Text level="small" weight={500} shade={9}>
        Create Range
      </Text.Text>
      <Text.Text level="h4" weight={450} shade={10}>
        {range.name}
      </Text.Text>
    </Align.Space>
    <Handle.Source location="right" id="output" />
  </Align.Pack>
);
