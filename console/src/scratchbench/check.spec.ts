import { writeFileSync } from "node:fs";
import { it } from "vitest";
import { z } from "zod";

import { createNextAIChannel } from "@/feature/ni/task/createChannel";
import { NI } from "@/feature/ni";

it("check", () => {
  const out: string[] = [];
  const formZ = z.object({ name: z.string(), config: NI.Task.analogReadConfigZ });
  const channels: any[] = [];
  for (let i = 0; i < 512; i++) channels.push({ ...createNextAIChannel(channels), device: "dev1" });
  const values = { name: "task", config: NI.Task.analogReadConfigZ.parse({ channels }) };
  out.push("types: " + [...new Set(values.config.channels.map((c: any) => c.type))].join(","));
  out.push("keys per channel: " + Object.keys(values.config.channels[0]).length);
  const bad = structuredClone(values);
  (bad.config.channels[511] as any).port = "x";
  const r = formZ.safeParse(bad);
  out.push("corrupt last channel -> " + (r.success ? "VALID (parse is not checking)" : r.error.issues.map((i) => i.path.join(".")).join(";")));
  const loops = 2000;
  const t0 = process.hrtime.bigint();
  for (let i = 0; i < loops; i++) formZ.safeParse(values);
  const t1 = process.hrtime.bigint();
  out.push("hrtime per parse, 512 channels: " + (Number(t1 - t0) / 1e6 / loops).toFixed(4) + " ms");
  const vs = Array.from({ length: loops }, () => structuredClone(values));
  const t2 = process.hrtime.bigint();
  for (let i = 0; i < loops; i++) formZ.safeParse(vs[i]);
  const t3 = process.hrtime.bigint();
  out.push("hrtime per parse, fresh objects each time: " + (Number(t3 - t2) / 1e6 / loops).toFixed(4) + " ms");
  writeFileSync("/private/tmp/claude-501/-Users-emilianobonilla-Desktop-synnaxlabs-synnax/e6928546-7632-4225-bc51-91fcdcddf6b4/scratchpad/bench-check.txt", out.join("\n"));
});
