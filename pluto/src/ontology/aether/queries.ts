import { ontology } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";

export interface RelationshipFluxStore
  extends flux.UnaryStore<string, ontology.Relationship> {}

export interface ResourceFluxStore extends flux.UnaryStore<string, ontology.Resource> {}

export interface SubStore extends flux.Store {
  relationships: RelationshipFluxStore;
  resources: ResourceFluxStore;
}

const RELATIONSHIP_SET_LISTENER: flux.ChannelListener<
  SubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: async ({ store, changed }) => {
    store.relationships.set(ontology.relationshipToString(changed), changed);
  },
};

const RELATIONSHIP_DELETE_LISTENER: flux.ChannelListener<
  SubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: async ({ store, changed }) => {
    store.relationships.delete(ontology.relationshipToString(changed));
  },
};

export const RELATIONSHIP_STORE_CONFIG: flux.UnaryStoreConfig<SubStore> = {
  listeners: [RELATIONSHIP_SET_LISTENER, RELATIONSHIP_DELETE_LISTENER],
};

const RESOURCE_SET_LISTENER: flux.ChannelListener<SubStore, typeof ontology.idZ> = {
  channel: ontology.RESOURCE_SET_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: async ({ store, changed, client }) => {
    const res = await client.ontology.retrieve(changed);
    store.resources.set(changed.key, res);
  },
};

const RESOURCE_DELETE_LISTENER: flux.ChannelListener<SubStore, typeof ontology.idZ> = {
  channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: async ({ store, changed }) => {
    store.resources.delete(changed.key);
  },
};

export const RESOURCE_STORE_CONFIG: flux.UnaryStoreConfig<SubStore> = {
  listeners: [RESOURCE_SET_LISTENER, RESOURCE_DELETE_LISTENER],
};
