// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client: Synnax = createTestClient();

/** Exports the project as a bundle and returns its zip bytes. */
const exportBundle = async (key: project.Key): Promise<Uint8Array<ArrayBuffer>> => {
  const stream = await client.projects.export(key, { encoding: "JSON" });
  return new Uint8Array(await new Response(stream).arrayBuffer());
};

describe("Project.importBundle", () => {
  it("imports a bundle and selects the created project", async () => {
    const src = await client.projects.create({
      name: uniqueName("import"),
      layout: {},
    });
    await client.logs.create(src.key, { name: "Metrics" });
    const bundle = await exportBundle(src.key);
    const { store } = await createConsoleWrapper({ client });
    await Project.importBundle(`${src.name}.zip`, bundle, { client, store });
    const imported = Session.Project.selectSelected(store.getState());
    expect(imported).not.toEqual(src.key);
    const proj = await client.projects.retrieve(imported);
    expect(proj.name).toEqual(src.name);
    const children = await client.ontology.retrieveChildren(
      project.ontologyID(imported),
    );
    expect(children.map(({ name }) => name)).toEqual(["Metrics"]);
  });
});
