// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { createTestStore, renderWithConsole, stubClipboardWriteText } from "@/testutil";

const client = createTestClient();

const renderCopyItem = async (
  ids: ontology.ID[],
  resources: ontology.Resource[],
): Promise<void> => {
  const store = await createTestStore();
  const props: Tree.ContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids }),
    state: createState(resources),
  };
  await renderWithConsole(
    <PMenu.Menu>
      <Tree.CopyPropertiesContextMenuItem {...props} />
    </PMenu.Menu>,
    { store },
  );
};

describe("CopyPropertiesContextMenuItem", () => {
  const chID = ontology.idZ.parse("channel:1");
  const properties = { rate: 25, virtual: false };
  const resource = createResource(chID, "my_channel", properties);
  let writeText: Mock;

  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("should copy the resource's properties as JSON when clicked", async () => {
    await renderCopyItem([chID], [resource]);
    fireEvent.click(screen.getByText("Copy properties"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(JSON.stringify(properties)),
    );
  });

  it("should render nothing when more than one resource is selected", async () => {
    const otherID = ontology.idZ.parse("channel:2");
    const other = createResource(otherID, "other_channel", {});
    await renderCopyItem([chID, otherID], [resource, other]);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });

  it("should render nothing when the selection is empty", async () => {
    await renderCopyItem([], []);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });
});
