// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Arc } from "@synnaxlabs/pluto";
import { type RenderResult, waitFor } from "@testing-library/react";
import { Suspense } from "react";
import { describe, expect, it } from "vitest";

import { Path } from "@/feature/arc/editor/Path";
import { createConsoleWrapper, renderSuspended, uniqueName } from "@/testutil";

const client = createTestClient();

const createDirectory = async (
  parent: arc.directory.Key,
): Promise<arc.directory.Directory> =>
  await client.arcs.directories.create({ parent, name: uniqueName("dir") });

const createArc = async (): Promise<arc.Arc> =>
  await client.arcs.create({
    name: uniqueName("arc"),
    type: "automation",
    source: { mode: "text" },
  });

const moveArc = async (a: arc.Arc, to: arc.directory.Key): Promise<void> =>
  await client.arcs.directories.move(
    arc.directory.resourceID(to),
    arc.resourceID(a.key),
  );

const renderPath = async (key: arc.Key): Promise<RenderResult> => {
  const { wrapper } = await createConsoleWrapper({ client });
  return await renderSuspended(
    <Suspense fallback={null}>
      <Arc.Suspended arcKey={key}>
        <Path />
      </Arc.Suspended>
    </Suspense>,
    { wrapper },
  );
};

describe("Path", () => {
  it("should show the directories below the root, then the Arc", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const mid = await createDirectory(root.key);
    const leaf = await createDirectory(mid.key);
    const a = await createArc();
    await moveArc(a, leaf.key);
    const { container } = await renderPath(a.key);
    await waitFor(() =>
      expect(container.textContent).toBe(`${mid.name}${leaf.name}${a.name}`),
    );
  });

  it("should show only the name of an Arc in the root", async () => {
    const a = await createArc();
    const { container } = await renderPath(a.key);
    await waitFor(() => expect(container.textContent).toBe(a.name));
  });

  it("should follow a rename of a directory above the Arc", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const dir = await createDirectory(root.key);
    const a = await createArc();
    await moveArc(a, dir.key);
    const { container } = await renderPath(a.key);
    await waitFor(() => expect(container.textContent).toBe(`${dir.name}${a.name}`));
    const name = uniqueName("renamed");
    await client.arcs.directories.rename(dir.key, name);
    await waitFor(() => expect(container.textContent).toBe(`${name}${a.name}`));
  });

  it("should follow a move of the Arc", async () => {
    const root = await client.arcs.directories.retrieveRoot();
    const from = await createDirectory(root.key);
    const to = await createDirectory(root.key);
    const a = await createArc();
    await moveArc(a, from.key);
    const { container } = await renderPath(a.key);
    await waitFor(() => expect(container.textContent).toBe(`${from.name}${a.name}`));
    await moveArc(a, to.key);
    await waitFor(() => expect(container.textContent).toBe(`${to.name}${a.name}`));
  });
});
