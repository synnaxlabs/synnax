import { ontology } from "@synnaxlabs/client";

import { type flux } from "@/flux/aether";

export interface RelationshipFluxStore
  extends flux.UnaryStore<string, ontology.Relationship> {}

export interface ResourceFluxStore extends flux.UnaryStore<string, ontology.Resource> {}

export const RELATIONSHIPS_FLUX_STORE_KEY = "relationships";
export const RESOURCES_FLUX_STORE_KEY = "resources";

interface SubStore extends flux.Store {
  [RELATIONSHIPS_FLUX_STORE_KEY]: RelationshipFluxStore;
  [RESOURCES_FLUX_STORE_KEY]: ResourceFluxStore;
}

const RELATIONSHIP_SET_LISTENER: flux.ChannelListener<
  SubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: ({ store, changed }) =>
    store.relationships.set(ontology.relationshipToString(changed), changed),
};

const RELATIONSHIP_DELETE_LISTENER: flux.ChannelListener<
  SubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: ({ store, changed }) =>
    store.relationships.delete(ontology.relationshipToString(changed)),
};

export const RELATIONSHIP_FLUX_STORE_CONFIG: flux.UnaryStoreConfig<SubStore> = {
  listeners: [RELATIONSHIP_SET_LISTENER, RELATIONSHIP_DELETE_LISTENER],
};

const RESOURCE_SET_LISTENER: flux.ChannelListener<SubStore, typeof ontology.idZ> = {
  channel: ontology.RESOURCE_SET_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: async ({ store, changed, client }) =>
    store.resources.set(changed.key, await client.ontology.retrieve(changed)),
};

const RESOURCE_DELETE_LISTENER: flux.ChannelListener<SubStore, typeof ontology.idZ> = {
  channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: ({ store, changed }) => store.resources.delete(changed.key),
};

export const RESOURCE_FLUX_STORE_CONFIG: flux.UnaryStoreConfig<SubStore> = {
  listeners: [RESOURCE_SET_LISTENER, RESOURCE_DELETE_LISTENER],
};
