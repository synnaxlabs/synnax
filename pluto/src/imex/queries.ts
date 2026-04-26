import { type imex, type ontology } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export interface FluxSubStore extends Flux.Store {}

export interface ImportParams {
  parent?: ontology.ID | null;
  envelopes: imex.Envelope | imex.Envelope[];
}

const RESOURCE_NAME = "resource";
export const VERBS: Flux.Verbs = {
  present: "import",
  past: "imported",
  participle: "importing",
};

export const { useUpdate: useImport } = Flux.createUpdate<ImportParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: VERBS,
  update: async ({ data, client }) => {
    await client.imex.import_(data.parent ?? null, data.envelopes);
    return data;
  },
});
