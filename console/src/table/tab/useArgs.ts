import { table } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import z from "zod";

const argsZ = z.object({ key: table.keyZ });

export const useArgs = Panel.createSelectContextTabArgs(argsZ);
