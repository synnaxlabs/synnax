import { type alamos } from "@synnaxlabs/alamos";
import { UnexpectedError, type channel } from "@synnaxlabs/client";

import { cache } from "@/telem/client/cache";

export const CACHE_BUFFER_SIZE = 10000;

export class CacheManager {
  private readonly ins: alamos.Instrumentation;
  private readonly cache = new Map<channel.Key, cache.Cache>();
  private readonly channelRetriever: channel.Retriever;

  constructor(ret: channel.Retriever, ins: alamos.Instrumentation) {
    this.ins = ins;
    this.channelRetriever = ret;
  }

  async populateMissing(keys: channel.Keys): Promise<void> {
    const toFetch: channel.Keys = [];
    for (const key of keys) {
      if (this.cache.has(key)) continue;
      toFetch.push(key);
    }
    if (toFetch.length === 0) return;
    const channels = await this.channelRetriever.retrieve(toFetch);
    for (const c of channels) {
      const ins = this.ins.child(`cache-${c.name}-${c.key}`);
      const cache_ = new cache.Cache(CACHE_BUFFER_SIZE, c, ins);
      if (!this.cache.has(c.key)) this.cache.set(c.key, cache_);
    }
  }

  get(key: channel.Key): cache.Cache {
    const c = this.cache.get(key);
    if (c != null) return c;
    throw new UnexpectedError(`cache entry for ${key} not found`);
  }

  close(): void {
    this.cache.forEach((c) => c.close());
    this.cache.clear();
  }
}
