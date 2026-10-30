import { writeFileSync } from "node:fs";
import { Channel } from "@synnaxlabs/pluto";
import { it } from "vitest";

it("skip", () => {
  const base = { name: "a", virtual: false, isIndex: false, index: 0, dataType: "float32", leaseholder: 0, internal: false, expression: "" };
  const fmt = (label: string, v: object) => {
    const r = Channel.formSchema.safeParse(v);
    return label + " -> " + (r.success ? "valid" : r.error.issues.map((i) => i.path.join(".") + ": " + i.message).join(" ; "));
  };
  writeFileSync("/private/tmp/claude-501/-Users-emilianobonilla-Desktop-synnaxlabs-synnax/e6928546-7632-4225-bc51-91fcdcddf6b4/scratchpad/bench-skip.txt", [
    fmt("name empty string", { ...base, name: "" }),
    fmt("name undefined (type error)", { ...base, name: undefined }),
    fmt("dataType wrong type", { ...base, dataType: 7 }),
  ].join("\n"));
});
