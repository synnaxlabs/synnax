import { Size } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";

const memoryState = z.object({
  used: z.number(),
  total: z.number(),
  display: z.boolean().optional().default(true),
});

interface Memory {
  usedJSHeapSize: number;
  jsHeapSizeLimit: number;
}

interface Performance {
  memory: Memory;
}

export class UsageTracker extends aether.Leaf<typeof memoryState> {
  interval?: ReturnType<typeof setInterval>;

  static readonly TYPE = "memoryUsage";
  static readonly z = memoryState;
  schema = UsageTracker.z;

  async afterUpdate(): Promise<void> {
    if (this.interval != null) return;
    const perf = performance as unknown as Performance;
    if (!("memory" in perf)) return this.setState((p) => ({ ...p, display: false }));
    this.interval = setInterval(() => {
      const memory = (perf as Performance).memory;
      this.setState({
        display: true,
        used: memory.usedJSHeapSize,
        total: memory.jsHeapSizeLimit,
      });
    }, 2000);
  }

  async afterDelete(): Promise<void> {
    if (this.interval != null) clearInterval(this.interval);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [UsageTracker.TYPE]: UsageTracker,
};
