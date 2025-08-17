import { ranger } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import { type flux } from "@/flux/aether";

export const FLUX_STORE_KEY = "ranges";

export interface FluxStore extends flux.UnaryStore<ranger.Key, ranger.Range> {}

interface SubStore extends flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}
const SET_LISTENER: flux.ChannelListener<SubStore, typeof ranger.payloadZ> = {
  channel: ranger.SET_CHANNEL_NAME,
  schema: ranger.payloadZ,
  onChange: ({ store, changed, client }) =>
    store.ranges.set(changed.key, client.ranges.sugarOne(changed)),
};

const DELETE_LISTENER: flux.ChannelListener<SubStore, typeof ranger.keyZ> = {
  channel: ranger.DELETE_CHANNEL_NAME,
  schema: ranger.keyZ,
  onChange: ({ store, changed }) => store.ranges.delete(changed),
};

export const STORE_CONFIG: flux.UnaryStoreConfig<SubStore, ranger.Key, ranger.Range> = {
  equal: (a, b) => deep.equal(a.payload, b.payload),
  listeners: [SET_LISTENER, DELETE_LISTENER],
};
