import { Icon } from "@synnaxlabs/media";

import { Align } from "@/align";
import { Icon as PIcon } from "@/icon";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

export const Symbol = () => (
  <Align.Pack
    x
    align="center"
    bordered
    borderShade={5}
    rounded={1}
    style={{ backgroundColor: "orange", width: "6rem" }}
    justify="center"
  >
    <PIcon.Icon
      style={{
        padding: "0.5rem",
        paddingBottom: "0.25rem",
      }}
    >
      <Icon.Select
        style={{
          width: "4rem",
          height: "4rem",
          color: "var(--pluto-gray-l0)",
        }}
      />
    </PIcon.Icon>
    <Handle.Sink location="left" id="value" />
    <Handle.Source location="right" id="true" style={{ top: "33%" }} />
    <Handle.Source location="right" id="false" style={{ top: "66%" }} />
  </Align.Pack>
);
