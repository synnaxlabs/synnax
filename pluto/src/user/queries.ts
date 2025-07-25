import { NotFoundError, type ontology, type user } from "@synnaxlabs/client";

import { Flux } from "@/flux";

interface RetrieveParams {
  key: user.Key;
}

export const retrieve = Flux.createRetrieve<RetrieveParams, user.User>({
  name: "User",
  retrieve: async ({ params, client }) => await client.user.retrieve(params.key),
});

interface RetrieveCreator {
  id: ontology.ID;
}

export const retrieveCreator = Flux.createRetrieve<RetrieveCreator, user.User>({
  name: "User",
  retrieve: async ({ params, client }) => {
    const user = await client.ontology.retrieve({
      ids: [params.id],
      creator: true,
    });
    if (user.length === 0)
      throw new NotFoundError(`No user with id ${params.id.key} found`);
    return await client.user.retrieve(user[0].id.key);
  },
});
