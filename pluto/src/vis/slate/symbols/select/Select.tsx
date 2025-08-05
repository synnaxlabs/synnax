import { color } from "@synnaxlabs/x";

import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Minimal } from "@/vis/slate/symbols/Base";

const ORANGE_HEX = color.construct("#FF8A00");

export const Symbol = () => (
  <Minimal
    sinks={[{ key: "value", Icon: Icon.Boolean }]}
    centerSinks
    sources={[
      { key: "true", Icon: Icon.True },
      { key: "false", Icon: Icon.False },
    ]}
  >
    <Flex.Box
      style={{
        width: "5rem",
        overflow: "hidden",
      }}
    >
      <Icon.Select
        style={{
          width: "7rem",
          height: "7rem",
          color: color.cssString(ORANGE_HEX),
          strokeWidth: "1px",
          transform: "translateX(-20%)",
        }}
      />
    </Flex.Box>
  </Minimal>
);
