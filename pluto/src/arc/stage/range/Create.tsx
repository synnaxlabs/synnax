import { ranger } from "@synnaxlabs/client";
import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Color } from "@/color";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

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
  <Base
    type="Create Range"
    Icon={<Icon.Range />}
    color="var(--pluto-warning-z-20)"
    textColor="var(--pluto-warning-z)"
    sinks={[
      {
        key: "to_do",
        Icon: Icon.ToDo,
      },
      {
        key: "in_progress",
        Icon: Icon.InProgress,
      },
      {
        key: "completed",
        Icon: Icon.Completed,
      },
    ]}
  >
    <Flex.Box direction="x" align="center" gap="small">
      {range.color != null && <Color.Swatch value={range.color} size="tiny" />}
      <Text.Text level="p" weight={500}>
        {range.name}
      </Text.Text>
    </Flex.Box>
  </Base>
);
