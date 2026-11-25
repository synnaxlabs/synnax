// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  group,
  panel,
  type resource,
  type Synnax as Client,
  view,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { DataType } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { Group } from "@/feature/group";
import { Modals } from "@/platform/modals";
import { Tree } from "@/platform/tree";
import { createTabWrapper, findToolbarIconButton } from "@/platform/view/testutil";
import {
  createTestClientWithGrants,
  getBySelector,
  queryIconButton,
  renderLazy,
} from "@/testutil";

const client = createTestClient();

const TAB = { variant: "view", type: Channel.Explorer.TAB_TYPE } as const;

const { Content: Explorer } = Channel.Explorer.TAB;

interface Folders {
  outer: group.Group;
  inner: group.Group;
  floats: channel.Channel;
  ints: channel.Channel;
}

/** Creates a float and an int channel in a folder that sits in another folder. */
const createFolders = async (): Promise<Folders> => {
  const outer = await client.groups.create({
    scope: "channel",
    name: uniqueName("outer"),
  });
  const inner = await client.groups.create({
    parent: group.resourceID(outer.key),
    name: uniqueName("inner"),
  });
  const [floats, ints] = await client.channels.create([
    {
      name: uniqueName("floats"),
      dataType: DataType.FLOAT32,
      virtual: true,
      group: inner.key,
    },
    {
      name: uniqueName("ints"),
      dataType: DataType.INT64,
      virtual: true,
      group: inner.key,
    },
  ]);
  return { outer, inner, floats, ints };
};

// The Folders layout shows the resource tree, which reads its items from the provider.
const TREE_ITEMS: Tree.Items = {
  ...Channel.TREE_ITEMS,
  ...Group.createTreeItems([Channel.GROUP_MEMBER]),
};

const TreeExplorer = (): ReactElement => (
  <Haul.Provider>
    <Tree.Provider items={TREE_ITEMS}>
      <Explorer />
      <Modals.Stack />
    </Tree.Provider>
  </Haul.Provider>
);

/**
 * @returns the elements with the text in the Folders layout, outside the full tree that
 * stays mounted and hidden under a search answer.
 */
const shownInFolders = (text: string): HTMLElement[] =>
  screen
    .queryAllByText(text)
    .filter(
      (el) =>
        el.closest(".console-channel-explorer__folders") != null &&
        el.closest(".console-tree--hidden") == null,
    );

const findBySelector = async (selector: string): Promise<HTMLElement> =>
  await waitFor(() => getBySelector(document.body, selector));

// The calculation editor's code field needs a provider these specs do not mount, and
// its boundary catches that error. Any other caught error logs, which fails the spec.
const ignoreCodeProvider = (error: unknown): void => {
  if (!String(error).includes("Code.Provider is not mounted")) console.error(error);
};

const renderView = async (stored: view.View, as?: Client): Promise<void> => {
  const { wrapper } = await createTabWrapper({
    client,
    as,
    tab: { variant: "resource", resource: view.resourceID(stored.key) },
  });
  await renderLazy(<TreeExplorer />, { wrapper, onCaughtError: ignoreCodeProvider });
};

describe("channel explorer", () => {
  it("should show the matches of a filter under their folders", async () => {
    const { outer, inner, floats, ints } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [inner.key], dataType: { in: [DataType.FLOAT32] } },
      }),
    );
    await waitFor(() => expect(shownInFolders(floats.name)).toHaveLength(1));
    expect(shownInFolders(inner.name)).toHaveLength(1);
    expect(shownInFolders(outer.name)).toHaveLength(1);
    expect(shownInFolders(ints.name)).toHaveLength(0);
  });

  it("should show a chip for each field of a stored filter", async () => {
    const { inner } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: {
          groups: [inner.key],
          dataType: { in: [DataType.FLOAT32] },
          virtual: true,
        },
      }),
    );
    const chips = within(await findBySelector(".console-view__chips"));
    expect(await chips.findByText(inner.name)).toBeTruthy();
    expect(chips.getByText("Folder")).toBeTruthy();
    expect(chips.getByText("Data type")).toBeTruthy();
    expect(chips.getByText("float32")).toBeTruthy();
    expect(chips.getByText("Kind")).toBeTruthy();
    expect(chips.getByText("Virtual")).toBeTruthy();
  });

  it("should list each channel with the folders that hold it", async () => {
    const { outer, inner, floats, ints } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [inner.key] },
        layout: "list",
      }),
    );
    expect(await screen.findByText(floats.name)).toBeTruthy();
    expect(screen.getByText(ints.name)).toBeTruthy();
    await waitFor(() =>
      expect(screen.getAllByText(`${outer.name} / ${inner.name}`)).toHaveLength(2),
    );
    expect(screen.getByText("int64")).toBeTruthy();
  });

  it("should offer to clear a filter that matches nothing, in each layout", async () => {
    const { outer } = await createFolders();
    for (const layout of view.CHANNEL_LAYOUTS) {
      await renderView(
        await client.views.create({
          name: uniqueName("view"),
          type: "channel",
          where: { groups: [outer.key] },
          layout,
        }),
      );
      expect(await screen.findByText("No channels match the filter")).toBeTruthy();
      fireEvent.click(screen.getByText("Clear filters"));
      await waitFor(() => expect(screen.queryByText("Folder")).toBeNull());
      cleanup();
    }
  });

  it("should offer the actions on a channel in the List layout", async () => {
    const { inner, floats } = await createFolders();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [inner.key] },
        layout: "list",
      }),
    );
    fireEvent.contextMenu(await screen.findByText(floats.name));
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Copy link")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
  });

  it("should save the layout of a stored view", async () => {
    const stored = await client.views.create({
      name: uniqueName("view"),
      type: "channel",
    });
    await renderView(stored);
    fireEvent.click(await screen.findByRole("button", { name: "List" }));
    await waitFor(async () => {
      const saved = await client.views.retrieve(stored.key);
      expect(saved.type === "channel" && saved.layout).toBe("list");
    });
  });
});

describe("channel inspector", () => {
  /** Creates a channel in a folder and a calculation that reads it. */
  const createRead = async () => {
    const folder = await client.groups.create({
      scope: "channel",
      name: uniqueName("folder"),
    });
    const read = await client.channels.create({
      name: uniqueName("read"),
      dataType: DataType.FLOAT32,
      virtual: true,
      group: folder.key,
    });
    const unused = await client.channels.create({
      name: uniqueName("unused"),
      dataType: DataType.FLOAT32,
      virtual: true,
      group: folder.key,
    });
    const reader = await client.channels.create({
      name: uniqueName("reader"),
      dataType: DataType.FLOAT32,
      virtual: true,
      expression: `return ${read.name} * 2`,
    });
    return { folder, read, unused, reader };
  };

  const renderList = async (folder: group.Group, as?: Client) =>
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [folder.key] },
        layout: "list",
      }),
      as,
    );

  const inspector = () => within(screen.getByLabelText("Channel inspector"));

  it("should show what reads the checked channel and open it", async () => {
    const { folder, read, reader } = await createRead();
    await renderList(folder);
    expect(
      await inspector().findByText("Select a channel to see what uses it."),
    ).toBeTruthy();
    fireEvent.click(await screen.findByText(read.name), { ctrlKey: true });
    expect(await inspector().findByText("Read by")).toBeTruthy();
    fireEvent.click(await inspector().findByText(reader.name));
    expect(await screen.findByDisplayValue(reader.name)).toBeTruthy();
  });

  it("should say that nothing uses an unused channel", async () => {
    const { folder, unused } = await createRead();
    await renderList(folder);
    fireEvent.click(await screen.findByText(unused.name), { ctrlKey: true });
    expect(await inspector().findByText("Nothing uses this channel.")).toBeTruthy();
  });

  it("should count the channels when several are checked", async () => {
    const { folder, read, unused } = await createRead();
    await renderList(folder);
    fireEvent.click(await screen.findByText(read.name), { ctrlKey: true });
    fireEvent.click(await screen.findByText(unused.name), { ctrlKey: true });
    expect(await inspector().findByText("2 channels selected")).toBeTruthy();
  });

  it("should follow the channel selected in the Folders layout", async () => {
    const { folder, read, reader } = await createRead();
    await renderView(
      await client.views.create({
        name: uniqueName("view"),
        type: "channel",
        where: { groups: [folder.key] },
      }),
    );
    const tree = within(await findBySelector(".console-channel-explorer__folders"));
    fireEvent.click(await tree.findByText(read.name));
    expect(await inspector().findByText(reader.name)).toBeTruthy();
  });

  describe("description", () => {
    /** Creates a channel with the description in a folder of its own. */
    const createDescribed = async (description: string) => {
      const folder = await client.groups.create({
        scope: "channel",
        name: uniqueName("folder"),
      });
      const ch = await client.channels.create({
        name: uniqueName("described"),
        dataType: DataType.FLOAT32,
        virtual: true,
        group: folder.key,
        description,
      });
      return { folder, ch };
    };

    const findEditor = async (): Promise<HTMLElement> =>
      await inspector().findByRole("textbox", { name: "Channel description" });

    const storedDescription = async (key: channel.Key): Promise<string> =>
      (await client.channels.retrieve(key)).description;

    it("should save a description added with Cmd+Enter", async () => {
      const { folder, ch } = await createDescribed("");
      await renderList(folder);
      fireEvent.click(await screen.findByText(ch.name), { ctrlKey: true });
      fireEvent.click(
        await inspector().findByRole("button", { name: "Add a description" }),
      );
      const editor = await findEditor();
      fireEvent.change(editor, { target: { value: "Fed by pump P-1." } });
      fireEvent.keyDown(editor, { key: "Enter", metaKey: true });
      expect(await inspector().findByText("Fed by pump P-1.")).toBeTruthy();
      expect(inspector().queryByRole("textbox")).toBeNull();
      await waitFor(async () =>
        expect(await storedDescription(ch.key)).toBe("Fed by pump P-1."),
      );
    });

    it("should drop the draft on Escape", async () => {
      const { folder, ch } = await createDescribed("Kept text");
      await renderList(folder);
      fireEvent.click(await screen.findByText(ch.name), { ctrlKey: true });
      fireEvent.click(await inspector().findByRole("button", { name: "Kept text" }));
      const editor = await findEditor();
      fireEvent.change(editor, { target: { value: "Dropped text" } });
      fireEvent.keyDown(editor, { key: "Escape" });
      const text = await inspector().findByRole("button", { name: "Kept text" });
      await waitFor(() => expect(document.activeElement).toBe(text));
      expect(inspector().queryByRole("textbox")).toBeNull();
      expect(await storedDescription(ch.key)).toBe("Kept text");
    });

    it("should open the editor again with the user's text when a save is refused", async () => {
      const { folder, ch } = await createDescribed("Kept text");
      await renderList(folder);
      fireEvent.click(await screen.findByText(ch.name), { ctrlKey: true });
      fireEvent.click(await inspector().findByRole("button", { name: "Kept text" }));
      // The field stops typing at the limit, so only a change event can pass it.
      const refused = "x".repeat(1025);
      fireEvent.change(await findEditor(), { target: { value: refused } });
      fireEvent.keyDown(await findEditor(), { key: "Enter", metaKey: true });
      await waitFor(async () =>
        expect((await findEditor()) as HTMLTextAreaElement).toHaveProperty(
          "value",
          refused,
        ),
      );
      expect(await storedDescription(ch.key)).toBe("Kept text");
    });

    it("should save a trimmed edit on blur", async () => {
      const { folder, ch } = await createDescribed("Old text");
      await renderList(folder);
      fireEvent.click(await screen.findByText(ch.name), { ctrlKey: true });
      fireEvent.click(await inspector().findByRole("button", { name: "Old text" }));
      const editor = await findEditor();
      fireEvent.change(editor, { target: { value: "  New text\n" } });
      fireEvent.blur(editor);
      await waitFor(async () =>
        expect(await storedDescription(ch.key)).toBe("New text"),
      );
    });

    const createReader = async () =>
      await createTestClientWithGrants(client, {
        retrieve: [
          channel.TYPE_RESOURCE_ID,
          view.TYPE_RESOURCE_ID,
          group.TYPE_RESOURCE_ID,
          panel.TYPE_RESOURCE_ID,
        ],
      });

    it("should show a subject who cannot update the channel the text and no editor", async () => {
      const { folder, ch } = await createDescribed("Visible text");
      await renderList(folder, await createReader());
      fireEvent.click(await screen.findByText(ch.name), { ctrlKey: true });
      fireEvent.click(await inspector().findByText("Visible text"));
      expect(inspector().queryByRole("button", { name: "Visible text" })).toBeNull();
      expect(inspector().queryByRole("textbox")).toBeNull();
    });

    it("should show a subject who cannot update the channel no empty description", async () => {
      const { folder, ch } = await createDescribed("");
      await renderList(folder, await createReader());
      fireEvent.click(await screen.findByText(ch.name), { ctrlKey: true });
      expect(await inspector().findByText(ch.name)).toBeTruthy();
      expect(inspector().queryByText("Add a description")).toBeNull();
    });
  });
});

describe("channel explorer permissions", () => {
  const createEditor = async (creatable: resource.ID[] = []) =>
    await createTestClientWithGrants(client, {
      retrieve: [
        channel.TYPE_RESOURCE_ID,
        view.TYPE_RESOURCE_ID,
        group.TYPE_RESOURCE_ID,
        panel.TYPE_RESOURCE_ID,
      ],
      update: [view.TYPE_RESOURCE_ID],
      create: [view.TYPE_RESOURCE_ID, ...creatable],
    });

  const renderExplorer = async (as: Client) => {
    const { wrapper } = await createTabWrapper({ client, as, tab: TAB });
    await renderLazy(<TreeExplorer />, { wrapper });
  };

  it("should withhold the create buttons from a subject who cannot create channels", async () => {
    await renderExplorer(await createEditor());
    await waitFor(() =>
      expect(
        queryIconButton(getBySelector(document.body, ".console-view__views"), "add"),
      ).toBeTruthy(),
    );
    expect(
      queryIconButton(getBySelector(document.body, ".console-view__toolbar"), "add"),
    ).toBeNull();
  });

  it("should offer the create button to a subject who may create channels", async () => {
    await renderExplorer(await createEditor([channel.TYPE_RESOURCE_ID]));
    expect(await findToolbarIconButton("add")).toBeTruthy();
  });
});
