import { log } from "@synnaxlabs/client";
import { Panel } from "@synnaxlabs/pluto";
import z from "zod";

const argsZ = z.object({ key: log.keyZ });

export const useArgs = Panel.createSelectContextTabArgs(argsZ);
