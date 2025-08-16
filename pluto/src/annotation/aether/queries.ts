import { annotation } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";
import { type ontology } from "@/ontology/aether";

export interface FluxStore
  extends flux.UnaryStore<annotation.Key, annotation.Annotation> {}

export interface SubStore extends flux.Store {
  annotations: FluxStore;
  relationships: ontology.RelationshipFluxStore;
}

const SET_ANNOTATION_LISTENER: flux.ChannelListener<
  SubStore,
  typeof annotation.annotationZ
> = {
  channel: annotation.SET_CHANNEL_NAME,
  schema: annotation.annotationZ,
  onChange: async ({ store, changed }) => {
    store.annotations.set(changed.key, changed);
  },
};

const DELETE_ANNOTATION_LISTENER: flux.ChannelListener<
  SubStore,
  typeof annotation.keyZ
> = {
  channel: annotation.DELETE_CHANNEL_NAME,
  schema: annotation.keyZ,
  onChange: async ({ store, changed }) => store.annotations.delete(changed),
};

export const STORE_CONFIG: flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_ANNOTATION_LISTENER, DELETE_ANNOTATION_LISTENER],
};
