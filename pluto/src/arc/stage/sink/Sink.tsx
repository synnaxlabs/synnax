import { channel } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import z from "zod/v4";

import { Base } from "@/arc/stage/Base";
import { type types } from "@/arc/stage/types";
import { Channel } from "@/channel";
import { Icon } from "@/icon";
import { Text } from "@/text";

export const config = z.object({
  channel: channel.keyZ,
});

export type Config = z.infer<typeof config>;

export interface SymbolProps extends types.SymbolProps<Config> {}

const FUCHSIA = color.construct("#FF00FF");
const FUCHSIA_BG = color.setAlpha(FUCHSIA, 0.2);
const FUCHSIA_TEXT = color.setAlpha(FUCHSIA, 1);

export const Symbol = ({ channel }: SymbolProps) => {
  const name =
    Channel.useRetrieve({ key: channel }, { addStatusOnFailure: false }).data?.name ??
    "Channel";

  return (
    <Base
      type="Sink"
      Icon={<Icon.Channel />}
      color={color.cssString(FUCHSIA_BG)}
      textColor={color.cssString(FUCHSIA_TEXT)}
      sinks={[{ key: "input", Icon: Icon.Value }]}
    >
      <Text.Text
        level="p"
        weight={500}
        color={10}
        style={{ maxWidth: 100 }}
        overflow="ellipsis"
      >
        {name}
      </Text.Text>
    </Base>
  );
};
