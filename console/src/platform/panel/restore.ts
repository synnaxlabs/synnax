// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  lineplot,
  log,
  type ontology,
  type project,
  schematic,
  type Synnax,
  table,
  UnexpectedError,
} from "@synnaxlabs/client";

export interface RestoreParams {
  client: Synnax;
  project: project.Key;
  corpse: unknown;
}

type Restorer = (params: RestoreParams) => Promise<void>;

// Corpses keep their original keys, so create() re-registers the document under
// the same key and every reference to it works again. Types absent from the map
// are not restorable; their tombstones offer Close only.
const RESTORERS: Record<string, Restorer> = {
  [schematic.TYPE_ONTOLOGY_ID.type]: async ({ client, project, corpse }) => {
    await client.schematics.create(project, corpse as schematic.New);
  },
  [lineplot.TYPE_ONTOLOGY_ID.type]: async ({ client, project, corpse }) => {
    await client.lineplots.create(project, corpse as lineplot.New);
  },
  [log.TYPE_ONTOLOGY_ID.type]: async ({ client, project, corpse }) => {
    await client.logs.create(project, corpse as log.New);
  },
  [table.TYPE_ONTOLOGY_ID.type]: async ({ client, project, corpse }) => {
    await client.tables.create(project, corpse as table.New);
  },
  [arc.TYPE_ONTOLOGY_ID.type]: async ({ client, corpse }) => {
    await client.arcs.create(corpse as arc.CreateParams);
  },
};

/** canRestore returns whether a deleted resource of the given type is restorable. */
export const canRestore = ({ type }: ontology.ID): boolean => type in RESTORERS;

/**
 * Restores a deleted document from its corpse under its original key.
 * @throws {UnexpectedError} if the resource type is not restorable; gate calls
 * with {@link canRestore}.
 */
export const restore = async (
  { type }: ontology.ID,
  params: RestoreParams,
): Promise<void> => {
  const restorer = RESTORERS[type];
  if (restorer == null)
    throw new UnexpectedError(`no restorer registered for resource type ${type}`);
  await restorer(params);
};
