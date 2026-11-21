import { it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { Schematic } from "@/feature/schematic";
import { renderWithConsole } from "@/testutil";

it("renders", async () => {
  void Schematic;
  void LinePlot;
  await renderWithConsole(<div>hi</div>);
  await new Promise((r) => setTimeout(r, 300));
});
