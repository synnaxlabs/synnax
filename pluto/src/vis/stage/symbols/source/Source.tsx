import { Icon } from "@synnaxlabs/media";

import { Align } from "@/align";
import { Text } from "@/text";
import { Handle } from "@/vis/stage/handle";

export const Symbol = () => (
  <Align.Space x>
    <Text.WithIcon startIcon={<Icon.Channel />} level="p">
      Telemetry Source
    </Text.WithIcon>
    <Handle.Source />
  </Align.Space>
);
