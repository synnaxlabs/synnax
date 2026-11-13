// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type group,
  project,
  query,
  type resource,
  type Synnax,
  UnexpectedError,
} from "@synnaxlabs/client";
import { array, deep, type record, verbs } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";

const RESOURCE_NAME = "project";

export type RetrieveQuery = {
  key: project.Key;
};

export const { use } = Flux.createRetrieve<RetrieveQuery, project.Project>({
  name: RESOURCE_NAME,
  retrieve: async ({ client, query }) => await client.projects.retrieve(query),
  onChange: ({ client, query }, handler) =>
    client.projects.watch(query).onChange(handler),
  getCached: ({ client, query }) => client.projects.watch(query).getCached(),
});

export type DeleteParams = project.Key | project.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams>({
  name: RESOURCE_NAME,
  verbs: verbs.DELETE,
  update: async ({ client, data, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    await client.projects.delete(keys, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export interface RenameParams {
  key: project.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams>({
  name: RESOURCE_NAME,
  verbs: verbs.RENAME,
  update: async ({ client, data, onOptimisticComplete }) => {
    const { key, name } = data;
    await client.projects.rename(key, name, {
      onOptimistic: async () => await onOptimisticComplete(data),
    });
    return data;
  },
});

export const formSchema = project.projectZ.partial({ key: true });

export const formPath = deep.path<z.infer<typeof formSchema>>();

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
};

export const useForm = Flux.createForm<RetrieveQuery, typeof formSchema>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ client, query: { key } }) => await client.projects.retrieve(key),
  getCached: ({ client, query: { key } }) => {
    const cached = client.projects.watch(key).getCached();
    return query.isLive(cached) ? cached : undefined;
  },
  update: async ({ client, value, set }) => {
    const res = await client.projects.create(value());
    set(formPath.key, res.key);
  },
});

/** A record type that belongs to a project. */
export type VisualizationType = "schematic" | "lineplot" | "log" | "table" | "panel";

/** A visualization of any type. */
export interface Visualization extends record.KeyedNamed<string>, project.Movable {}

/** The flat where fields that every visualization client takes. */
export interface VisualizationQuery {
  key?: string[];
  groups?: group.Key[];
  grouped?: boolean;
  projects?: project.Key[];
}

/** The calls that every visualization client answers. */
export interface VisualizationClient {
  retrieve(query: VisualizationQuery): Promise<Visualization[]>;
  watch(
    query: VisualizationQuery & { order?: { field: "name" } },
  ): query.List<Visualization, unknown>;
  readonly setProjectRequest: project.MoveRequest<string>;
}

/** Returns the client of each visualization type. */
export const VISUALIZATIONS: Record<
  VisualizationType,
  (client: Synnax) => VisualizationClient
> = {
  schematic: (client) => client.schematics,
  lineplot: (client) => client.lineplots,
  log: (client) => client.logs,
  table: (client) => client.tables,
  panel: (client) => client.panels,
};

export const VISUALIZATION_TYPES = Object.keys(VISUALIZATIONS) as VisualizationType[];

export const isVisualizationType = (type: resource.Type): type is VisualizationType =>
  type in VISUALIZATIONS;

export type RetrieveSiblingsQuery = {
  resourceID?: resource.ID;
  types: VisualizationType[];
};

export interface Sibling extends record.KeyedNamed<string> {
  type: VisualizationType;
}

/**
 * Retrieves the visualizations of the given types that share the queried
 * visualization's project, excluding the visualization itself. Returns an empty list
 * when the visualization belongs to no project.
 * @throws if the resource is not a visualization.
 */
export const retrieveSiblings = async (
  client: Synnax,
  { resourceID, types }: RetrieveSiblingsQuery,
): Promise<Sibling[]> => {
  if (resourceID == null) return [];
  if (!isVisualizationType(resourceID.type))
    throw new UnexpectedError(`a ${resourceID.type} does not belong to a project`);
  const [source] = await VISUALIZATIONS[resourceID.type](client).retrieve({
    key: [resourceID.key],
  });
  if (source?.project == null) return [];
  const projects = [source.project];
  const siblings = await Promise.all(
    types.map(async (type) =>
      (await VISUALIZATIONS[type](client).retrieve({ projects })).map(
        ({ key, name }) => ({ key, name, type }),
      ),
    ),
  );
  return siblings.flat().filter(({ key }) => key !== resourceID.key);
};
