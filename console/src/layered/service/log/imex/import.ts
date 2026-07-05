// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, log, project } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { errors } from "@synnaxlabs/x";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/layered/service/log/layout";

// The Core owns log envelope decoding and legacy-version migration, so the file's bytes
// are streamed up untouched and the imported log is read back to populate local state.
// The Core import path does not yet parent the log to a project, so the project ->
// ParentOf -> log relationship is wired here to mirror a regular log create.
const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.createGranted({ id: log.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import logs");
  if (client == null) throw new DisconnectedError();
  const id = await client.imex.import(JSON.stringify(data), { encoding: "JSON" });
  try {
    await client.ontology.addChildren(project.ontologyID(projectKey), id);
  } catch (err) {
    try {
      await client.logs.delete(id.key);
    } catch (deleteErr) {
      console.error("failed to delete orphaned log after import failure", deleteErr);
    }
    throw errors.fromUnknown(err);
  }
  placeLayout(create({ ...layout, key: id.key }));
  return id;
};

export const FILE_INGESTERS: Import.FileIngesters = { [LAYOUT_TYPE]: ingest };
