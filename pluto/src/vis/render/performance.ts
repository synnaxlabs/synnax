import { Destructor, TimeSpan } from "@synnaxlabs/x";

class TrackerEntry {
  level: number;
  private total: number;
  private overTarget: number;
  private readonly target: TimeSpan;

  constructor(target: TimeSpan) {
    this.target = target;
    this.overTarget = 0;
    this.level = 0;
    this.total = 0;
  }

  measure(): [number, Destructor] {
    const start = performance.now();
    return [
      this.level,
      () => {
        const elapsed = TimeSpan.milliseconds(performance.now() - start);
        if (elapsed.greaterThan(this.target)) this.overTarget++;
        this.total++;
      },
    ];
  }

  updateLevel(): void {
    if (this.total === 0 || this.overTarget === 0) return;
    const overTargetFrac = this.overTarget / this.total;
    if (overTargetFrac < 0.125)
      if (this.level > 0) this.level--;
      else if (overTargetFrac > 0.25) this.level++;
  }
}

export class Tracker {
  private readonly entries: Map<string, TrackerEntry>;
  private readonly target: TimeSpan;

  constructor(target: TimeSpan) {
    this.entries = new Map();
    this.target = target;
  }

  measure(key: string): [number, Destructor] {
    if (!this.entries.has(key)) this.entries.set(key, new TrackerEntry(this.target));
    return this.entries.get(key)!.measure();
  }

  updateLevels(): void {
    for (const entry of this.entries.values()) entry.updateLevel();
  }

  levels(): Map<string, number> {
    const levels = new Map<string, number>();
    for (const [key, entry] of this.entries) levels.set(key, entry.level);
    return levels;
  }
}
