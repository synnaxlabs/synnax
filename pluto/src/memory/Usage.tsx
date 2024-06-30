import { Size } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { memory } from "@/memory/aether";
import { Text } from "@/text";

export const Usage = Aether.wrap(memory.UsageTracker.TYPE, ({ aetherKey }) => {
  const [, { used, total, display }] = Aether.use({
    aetherKey,
    type: memory.UsageTracker.TYPE,
    schema: memory.UsageTracker.z,
    initialState: { used: 0, total: 0 },
  });
  return (
    <Text.Text level="p" style={{ padding: "0 2rem" }}>
      {display
        ? `${Size.bytes(used).truncate(Size.MEGABYTE).toString()} / ${Size.bytes(total).truncate(Size.MEGABYTE).toString()}`
        : "N/A"}
    </Text.Text>
  );
});
