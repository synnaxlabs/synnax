import { writeFileSync } from "node:fs";
import { it } from "vitest";

import { createNextAIChannel } from "@/feature/ni/task/createChannel";
import { NI } from "@/feature/ni";
import { OPCUA } from "@/feature/opcua";

const show = (r: any) =>
  r.success ? "valid" : r.error.issues.map((i: any) => `${i.path.join(".")} [${i.params?.variant ?? "error"}]: ${i.message}`).join(" ; ");

it("bugs", () => {
  const out: string[] = [];
  const base = OPCUA.Task.deployReadConfigZ;
  const ch = (k: string, isIndex = false) => ({ key: k, nodeId: "ns=2;i=5", nodeName: "n", channel: 0, enabled: true, useAsIndex: false, isIndex, dataType: "float32", name: k });
  const opc = { device: "dev1", sampleRate: 10, streamRate: 5, arrayMode: false, arraySize: 1, dataSaving: true, autoStart: false, channels: [ch("a", true), ch("b", true)] };
  const parsed = OPCUA.Task.READ_SCHEMAS.config.safeParse(opc);
  out.push("opc shape parse: " + show(parsed));
  out.push("opc deploy: " + show(base.safeParse(parsed.success ? parsed.data : opc)));
  const ai: any = { ...createNextAIChannel([]), device: "dev1" };
  ai.customScale = { type: "map", preScaledMin: 10, preScaledMax: 0, scaledMin: 5, scaledMax: 1, preScaledUnits: "Volts", scaledUnits: "Volts" };
  const cfg = NI.Task.analogReadConfigZ.parse({ channels: [ai], sampleRate: 10, streamRate: 5 });
  out.push("ni scale in config: " + JSON.stringify(cfg.channels[0].customScale));
  out.push("ni form schema: " + show(NI.Task.analogReadConfigZ.safeParse(cfg)));
  out.push("ni deploy schema: " + show(NI.Task.deployAnalogReadConfigZ.safeParse(cfg)));
  out.push("ni console scale schema alone: " + show(NI.Task.SCALE_SCHEMAS.map.safeParse(cfg.channels[0].customScale)));
  writeFileSync("/private/tmp/claude-501/-Users-emilianobonilla-Desktop-synnaxlabs-synnax/e6928546-7632-4225-bc51-91fcdcddf6b4/scratchpad/bench-bugs.txt", out.join("\n"));
});
