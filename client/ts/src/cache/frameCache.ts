import { Frame } from "@/framer";
import { Size, TimeRange } from "@/telem";

export class FrameCache {
  private readonly _cache: Record<string, Frame>;

  constructor() {
    this._cache = {};
  }

  size(): Size {
    return Object.values(this._cache).reduce(
      (acc, fr) => acc.add(fr.size()),
      Size.ZERO
    );
  }

  get(tr: TimeRange, ...keys: string[]): FrameCacheResult {
    const strKey = tr.toString();
    const fr = this._cache[strKey];
    if (fr == null) return { frame: new Frame(), missing: keys };
    const filtered = fr.getF(keys);
    return { frame: filtered, missing: keys.filter((key) => !filtered.has(key)) };
  }

  set(tr: TimeRange, fr: Frame): void {
    const v = this._cache[this.key(tr)];
    this._cache[this.key(tr)] = v.overrideF(fr);
  }

  private key(tr: TimeRange): string {
    return tr.toString();
  }
}

export interface FrameCacheResult {
  frame: Frame;
  missing: string[];
}
