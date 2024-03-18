import { type alamos } from "@synnaxlabs/alamos";
import { TimeSpan, UnexpectedError, type channel } from "@synnaxlabs/client";
import { type Required } from "@synnaxlabs/x";

import { Unary } from "@/telem/client/cache/unary";

export const CACHE_BUFFER_SIZE = 10000;

/** Props for instantiating an @see Cache */
export interface CacheProps {
  /** For logging purposes */
  instrumentation: alamos.Instrumentation;
  /** Used to populate new cache entries with relevant info about the channel */
  channelRetriever: channel.Retriever;
  /**
   * Sets the size of the buffer in the dynamic cache
   * TODO: At some point this value should be calculated dynamically using heuristics
   * @default 10000
   */
  dynamicBufferSize?: number;
  /**
   * Sets the interval at which the cache will garbage collect, removing data that
   * currently in use by the rest of hte program.
   * @default TimeSpan.seconds(30)
   */
  gcInterval?: TimeSpan;
}

export class Cache {
  private readonly props: Required<CacheProps>;
  private readonly cache = new Map<channel.Key, Unary>();
  private readonly gcInterval: ReturnType<typeof setInterval>;

  constructor(props: CacheProps) {
    this.props = {
      dynamicBufferSize: CACHE_BUFFER_SIZE,
      gcInterval: TimeSpan.seconds(30),
      ...props,
    };
    this.gcInterval = setInterval(
      () => this.garbageCollect(),
      this.props.gcInterval.milliseconds,
    );
  }

  async populateMissing(keys: channel.Keys): Promise<void> {
    const { instrumentation: ins, channelRetriever, dynamicBufferSize } = this.props;
    const toFetch: channel.Keys = [];
    for (const key of keys) if (!this.cache.has(key)) toFetch.push(key);
    if (toFetch.length === 0) return;
    const channels = await channelRetriever.retrieve(toFetch);
    for (const c of channels) {
      const unaryIns = ins.child(`cache-${c.name}-${c.key}`);
      const unary = new Unary(dynamicBufferSize, c, unaryIns);
      if (!this.cache.has(c.key)) this.cache.set(c.key, unary);
    }
  }

  get(key: channel.Key): Unary {
    const c = this.cache.get(key);
    if (c != null) return c;
    throw new UnexpectedError(`cache entry for ${key} not found`);
  }

  private garbageCollect(): void {
    this.cache.forEach((c) => c.garbageCollect());
  }

  close(): void {
    clearInterval(this.gcInterval);
    this.cache.forEach((c) => c.close());
    this.cache.clear();
  }
}
