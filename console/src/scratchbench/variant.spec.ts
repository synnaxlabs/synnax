import { writeFileSync } from "node:fs";
import { zod } from "@synnaxlabs/x";
import { it } from "vitest";
import { z } from "zod";

it("variant", () => {
  const common = { name: z.string().min(1, "Name is required"), dataType: z.string() };
  const schema = z.discriminatedUnion("isIndex", [
    z.object({ ...common, isIndex: z.literal(true), virtual: z.literal(false) }),
    z.discriminatedUnion("virtual", [
      z.object({ ...common, isIndex: z.literal(false), virtual: z.literal(true) }),
      z.object({
        ...common,
        isIndex: z.literal(false),
        virtual: z.literal(false),
        index: z.number().refine((k) => k !== 0, "Data channel must have an index"),
      }),
    ]),
  ]);
  const out: string[] = [];
  for (const [label, v] of [
    ["data", { name: "a", dataType: "float32", isIndex: false, virtual: false, index: 0 }],
    ["virtual", { name: "a", dataType: "float32", isIndex: false, virtual: true, index: 0 }],
    ["index", { name: "a", dataType: "timestamp", isIndex: true, virtual: false, index: 0 }],
  ] as const) {
    const f = zod.getFieldSchema(schema, "index", { optional: true, values: v });
    const r = schema.safeParse(v);
    out.push(
      label + ": index field " + (f == null ? "absent" : "present, empty 0 refused=" + !f.safeParse(0).success) +
        " | parse " + (r.success ? "ok, output keys " + Object.keys(r.data).join(",") : r.error.issues.map((i) => i.path.join(".") + ": " + i.message).join(";")),
    );
  }
  writeFileSync("/private/tmp/claude-501/-Users-emilianobonilla-Desktop-synnaxlabs-synnax/e6928546-7632-4225-bc51-91fcdcddf6b4/scratchpad/bench-variant.txt", out.join("\n"));
});
