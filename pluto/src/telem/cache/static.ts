import { TimeRange } from "@synnaxlabs/x";

import { VisArray } from "@/telem/visArray";

export class StaticCache {
  private readonly entries: CachedRead[];

  constructor() {
    this.entries = [];
  }

  extent(): TimeRange {
    if (this.entries.length === 0) return TimeRange.ZERO;
    const first = this.entries[0].timeRange;
    const last = this.entries[this.entries.length - 1].timeRange;
    return new TimeRange(first.start, last.end);
  }

  gaps(): TimeRange[] {
    return this.entries.map((r) => r.gap);
  }

  write(tr: TimeRange, entries: VisArray[]): void {
    const read = new CachedRead(tr, entries);
    const i = this.getInsertionIndex(tr);
    if (i !== this.entries.length) {
      read.gap = new TimeRange(this.entries[i].timeRange.end, tr.end);
    }
    if (i !== 0) {
      const prev = this.entries[i - 1];
      prev.gap = new TimeRange(prev.timeRange.end, tr.start);
    }
    this.entries.splice(i, 0, new CachedRead(tr, entries));
  }

  private getInsertionIndex(tr: TimeRange): number {
    let i = 0;
    while (i < this.entries.length && this.entries[i].timeRange.start < tr.start) i++;
    return i;
  }

  read(tr: TimeRange): [VisArray[], TimeRange[]] {
    const reads = this.entries.filter((r) => {
      return r.timeRange.overlapsWith(tr);
    });
    if (reads.length === 0) return [[], [tr]];
    const gaps = reads
      .map((r) => r.gap)
      .filter((t, i) => i !== reads.length - 1 && !t.isZero);
    const leadingGap = new TimeRange(tr.start, reads[0].timeRange.start);
    const trailingGap = new TimeRange(reads[reads.length - 1].timeRange.end, tr.end);
    if (leadingGap.isValid && !leadingGap.isZero) gaps.unshift(leadingGap);
    if (trailingGap.isValid && !trailingGap.isZero) gaps.push(trailingGap);
    return [
      reads.flatMap((r) => r.data).filter((d) => d.timeRange.overlapsWith(tr)),
      gaps,
    ];
  }
}

class CachedRead {
  timeRange: TimeRange;
  data: VisArray[];
  gap: TimeRange;

  constructor(timeRange: TimeRange, data: VisArray[]) {
    this.timeRange = timeRange;
    this.data = data;
    this.gap = TimeRange.ZERO;
  }
}
