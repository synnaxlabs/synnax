// Scratch only: never commit.
import { Channel } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

import { createNextAIChannel } from "@/feature/ni/task/createChannel";
import { NI } from "@/feature/ni";

const time = (fn: () => void, runs: number): number => {
  for (let i = 0; i < Math.min(50, runs); i++) fn();
  const samples: number[] = [];
  for (let i = 0; i < runs; i++) {
    const t0 = performance.now();
    fn();
    samples.push(performance.now() - t0);
  }
  samples.sort((a, b) => a - b);
  return samples[Math.floor(samples.length / 2)];
};

export const run = (label: string): string => {
  const out: string[] = [`== ${label}`];
  const formZ = z.object({ name: z.string(), config: NI.Task.analogReadConfigZ });
  const deployZ = z.object({ name: z.string(), config: NI.Task.deployAnalogReadConfigZ });
  for (const n of [1, 8, 32, 128, 512]) {
    const channels: any[] = [];
    for (let i = 0; i < n; i++) channels.push({ ...createNextAIChannel(channels), device: "dev1" });
    const values = { name: "task", config: NI.Task.analogReadConfigZ.parse({ channels }) };
    if (!formZ.safeParse(values).success) out.push(`n=${n} form parse failed`);
    const runs = n >= 128 ? 200 : 1000;
    const full = time(() => formZ.safeParse(values), runs);
    const dep = time(() => deployZ.safeParse(values), runs);
    // One probe: copy the values with one channel field emptied, then parse.
    const probe = time(() => {
      const v = deep.copy(values);
      deep.set(v, "config.channels.0.port", 0);
      formZ.safeParse(v);
    }, runs);
    // Path copy: share every untouched branch, as the form state's own writes do.
    const pathProbe = time(() => {
      const chs = values.config.channels.slice();
      chs[0] = { ...chs[0], port: 0 };
      formZ.safeParse({ ...values, config: { ...values.config, channels: chs } });
    }, runs);
    out.push(`n=${String(n).padStart(3)} path-copy probe ${pathProbe.toFixed(3)} ms`);
    out.push(
      `n=${String(n).padStart(3)} channels: form parse ${full.toFixed(3)} ms | deploy parse ${dep.toFixed(3)} ms | one probe (copy + parse) ${probe.toFixed(3)} ms`,
    );
  }
  const base = {
    name: "",
    virtual: false,
    isIndex: false,
    index: 0,
    dataType: "float32",
    leaseholder: 0,
    internal: false,
    expression: "",
  };
  const fmt = (v: object) => {
    const r = Channel.formSchema.safeParse(v);
    return r.success ? "valid" : r.error.issues.map((i) => `${i.path.join(".")}: ${i.message}`).join(" ; ");
  };
  out.push(`channel form, name empty, index 0 -> ${fmt(base)}`);
  out.push(`channel form, name set, index 0  -> ${fmt({ ...base, name: "a" })}`);
  return out.join("\n");
};
