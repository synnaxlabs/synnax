import { createTestClient } from "@synnaxlabs/client/testutil";
import { it } from "vitest";

const client = createTestClient();

it("reads", async () => {
  await client.channels.retrieve({ internal: true, limit: 5 });
  const s = await client.openStreamer([]);
  s.close();
  await new Promise((r) => setTimeout(r, 300));
});
