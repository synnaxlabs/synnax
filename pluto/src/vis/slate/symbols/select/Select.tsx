import { Icon } from "@synnaxlabs/media";
import { color } from "@synnaxlabs/x";

import { Align } from "@/align";
import { Divider } from "@/divider";
import { Text } from "@/text";
import { Handle } from "@/vis/slate/handle";

const ORANGE_HEX = color.construct("#FF8A00");

export const Symbol = () => (
  <Align.Pack x align="center" bordered background={1} borderShade={6} rounded={1}>
    <Align.Space
      style={{
        borderTopLeftRadius: "1rem",
        borderBottomLeftRadius: "1rem",
        backgroundColor: color.cssString(color.setAlpha(ORANGE_HEX, 0.2)),
        width: "5rem",
        overflow: "hidden",
      }}
    >
      <Icon.Select
        style={{
          width: "8rem",
          height: "8rem",
          color: color.cssString(ORANGE_HEX),
          strokeWidth: "1px",
          transform: "translateX(-27%)",
        }}
      />
    </Align.Space>
    <Divider.Divider y shade={5} />
    <Align.Space style={{ padding: "0rem 1.5rem" }} align="start" size="tiny">
      <Text.Text
        level="small"
        weight={500}
        shade={9}
        style={{
          position: "relative",
          bottom: "0.25rem",
        }}
      >
        True
      </Text.Text>
      <Text.Text
        level="small"
        weight={500}
        shade={9}
        style={{
          position: "relative",
          top: "0.25rem",
        }}
      >
        False
      </Text.Text>
    </Align.Space>
    <Handle.Sink location="left" id="value" />
    <Handle.Source location="right" id="true" style={{ top: "30%" }} />
    <Handle.Source location="right" id="false" style={{ top: "70%" }} />
  </Align.Pack>
);
