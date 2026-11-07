// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { Tree } from "@/platform/tree";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { createTestStore, renderWithConsole } from "@/testutil";

const client = createTestClient();

const renderCopyItem = async (
  ids: resource.ID[],
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
  const chID = resource.idZ.parse("channel:1");
  const properties = { rate: 25, virtual: false };
  const ch = createResource(chID, "my_channel", properties);
  let writeText: Mock;

  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("should copy the resource's properties as JSON when clicked", async () => {
    await renderCopyItem([chID], [ch]);
    fireEvent.click(screen.getByText("Copy properties"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(JSON.stringify(properties)),
    );
  });

  it("should copy properties that hold a timestamp", async () => {
    const rackID = resource.idZ.parse("rack:65537");
    const rack = createResource(rackID, "my_rack", {
      key: 65537,
      status: { time: new TimeStamp(1000n) },
    });
    await renderCopyItem([rackID], [rack]);
    fireEvent.click(screen.getByText("Copy properties"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith('{"key":65537,"status":{"time":"1000"}}'),
    );
  });

  it("should render nothing when more than one resource is selected", async () => {
    const otherID = resource.idZ.parse("channel:2");
    const other = createResource(otherID, "other_channel", {});
    await renderCopyItem([chID, otherID], [ch, other]);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });

  it("should render nothing when the tree no longer holds the resource", async () => {
    await renderCopyItem([chID], []);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });

  it("should render nothing when the selection is empty", async () => {
    await renderCopyItem([], []);
    expect(screen.queryByText("Copy properties")).toBeNull();
  });
});
