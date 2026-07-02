// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, ontology } from "@synnaxlabs/client";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/platform/ontology";
import {
  buildBaseProps,
  buildResource,
  buildSelection,
  buildState,
} from "@/platform/ontology/testutil";
import { createTestStore, renderWithConsole } from "@/testutil";

const client = createTestClient();

const renderCopyItem = async (
  ids: ontology.ID[],
  resources: ontology.Resource[],
): Promise<void> => {
  const store = await createTestStore();
  const props: Ontology.TreeContextMenuProps = {
    ...buildBaseProps({ client, store }),
    selection: buildSelection({ ids }),
    state: buildState(resources),
  };
  await renderWithConsole(
    <PMenu.Menu>
      <Ontology.CopyPropertiesContextMenuItem {...props} />
    </PMenu.Menu>,
    { store },
  );
};

describe("CopyPropertiesContextMenuItem", () => {
  const chID = ontology.idZ.parse("channel:1");
  const resource = buildResource(chID, "my-channel", { rate: 25, virtual: false });

  it("should render a copy item for a single selection", async () => {
    await renderCopyItem([chID], [resource]);
    await waitFor(() => expect(screen.getByText("Copy properties")).toBeTruthy());
  });

  it("should render nothing when more than one resource is selected", async () => {
    const otherID = ontology.idZ.parse("channel:2");
    const other = buildResource(otherID, "other-channel", {});
    await renderCopyItem([chID, otherID], [resource, other]);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });

  it("should render nothing when the selection is empty", async () => {
    await renderCopyItem([], []);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });
});
