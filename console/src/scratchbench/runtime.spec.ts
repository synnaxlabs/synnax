import { writeFileSync } from "node:fs";
import { it } from "vitest";
import { z } from "zod";

z.config({ jitless: true });

import { run } from "@/scratchbench/body";

it("bench runtime", () => {
  writeFileSync("/private/tmp/claude-501/-Users-emilianobonilla-Desktop-synnaxlabs-synnax/e6928546-7632-4225-bc51-91fcdcddf6b4/scratchpad/bench-runtime.txt", run("zod/compile off"));
});
