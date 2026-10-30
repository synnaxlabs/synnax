import "zod/compile";

import { writeFileSync } from "node:fs";
import { it } from "vitest";

import { run } from "@/scratchbench/body";

it("bench compiled", () => {
  writeFileSync("/private/tmp/claude-501/-Users-emilianobonilla-Desktop-synnaxlabs-synnax/e6928546-7632-4225-bc51-91fcdcddf6b4/scratchpad/bench-compiled.txt", run("zod/compile on"));
});
