import { lineplot } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import z from "zod";

const argsZ = z.object({ key: lineplot.keyZ });

export const useArgs = Panel.createSelectContextTabArgs(argsZ);
