import { it } from "vitest";

it("runs", async () => {
  await new Promise((r) => setTimeout(r, 300));
});
