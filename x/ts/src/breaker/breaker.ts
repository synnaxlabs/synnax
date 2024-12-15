import { sleep } from "@/sleep";
import { CrudeTimeSpan, TimeSpan } from "@/telem";

export interface BreakerOptions {
  interval: CrudeTimeSpan;
  maxRetries: number;
  scale: number;
}

export class Breaker {
  private interval: TimeSpan;
  private maxRetries: number;
  private scale: number;

  constructor(options: BreakerOptions) {
    this.interval = new TimeSpan(options.interval);
    this.maxRetries = options.maxRetries;
    this.scale = options.scale;
  }

  public async run<F extends (...args: any[]) => any>(
    func: F,
    ...args: Parameters<F>
  ): Promise<ReturnType<F>> {
    let retries = 0;
    while (retries < this.maxRetries) {
      try {
        return await func(...args);
      } catch (err) {
        retries++;
        await sleep.sleep(this.interval);
        this.interval = this.interval.mult(this.scale);
      }
    }
    throw new Error("Max retries exceeded");
  }
}
