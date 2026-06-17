import { schematic } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import z from "zod";

const argsZ = z.object({ key: schematic.keyZ });

export const useArgs = Panel.createSelectContextTabArgs(argsZ);
