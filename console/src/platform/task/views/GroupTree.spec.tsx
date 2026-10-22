// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

interface Field extends Task.Views.GroupField {
  channel: number;
  name: string;
}

interface Group extends Task.Views.Group<Field> {
  label: string;
}

interface Leaf extends Task.Views.LeafGroup {
  label: string;
}

type Item = Group | Leaf;

const field = (key: string, pointer: string, channel = 0): Field => ({
  key,
  pointer,
  channel,
  name: "",
  disabled: false,
});

const groupLabel = Component.renderProp(({ groupKey }: Task.Views.GroupProps) => (
  <span>group-{groupKey}</span>
));

const groupDetails = Component.renderProp(({ path }: Task.Views.GroupProps) => (
  <span>details-for-{path}</span>
));

const fieldDetails = Component.renderProp(({ path }: Task.Views.FieldProps) => (
  <span>details-for-{path}</span>
));

const KINDS: Task.Views.GroupKind<Item>[] = [
  {
    name: "group",
    create: () => ({ key: id.create(), label: "", index: "", fields: [] }),
  },
  {
    name: "leaf",
    create: () => ({ key: id.create(), label: "", channel: 0, name: "", index: 0 }),
  },
];

const createField = (): Omit<Field, "key"> => field("", "");

const renderTree = (
  groups: Item[],
  props: Partial<Task.Views.GroupTreeProps<Item, Field>> = {},
) =>
  renderInTaskForm(
    <Task.Views.GroupTree<Item, Field>
      listTitle="Groups"
      emptyMessage="No groups"
      placeholder="Select a group or field"
      groupKinds={KINDS}
      createField={createField}
      groupLabel={groupLabel}
      groupDetails={groupDetails}
      fieldDetails={fieldDetails}
      {...props}
    />,
    { values: { config: { groups } } },
  );

const GROUP: Group = {
  key: "g",
  label: "",
  index: "ts",
  fields: [field("ts", "/time", 1), field("v", "/value", 2)],
};

const LEAF: Leaf = { key: "l", label: "", channel: 5, name: "flow", index: 4 };

describe("GroupTree", () => {
  it("should list a group's fields beneath it and hide its index field", async () => {
    await renderTree([GROUP]);
    await screen.findAllByText("group-g");
    expect(screen.getByText("/value")).toBeTruthy();
    expect(screen.queryByText("/time")).toBeNull();
  });

  it("should show the details of the selected field", async () => {
    await renderTree([GROUP]);
    fireEvent.click(await screen.findByText("/value"));
    await screen.findByText("details-for-config.groups.g.fields.v");
  });

  it("should add a group and select it", async () => {
    const { form } = await renderTree([]);
    fireEvent.click(await screen.findByRole("button", { name: "Add group" }));
    await waitFor(() =>
      expect(form.current?.get("config.groups").value).toHaveLength(1),
    );
    const [added] = form.current?.get<Group[]>("config.groups").value ?? [];
    await screen.findByText(`details-for-config.groups.${added.key}`);
  });

  it("should add every group a kind resolves with and select the first", async () => {
    const resolved: Group[] = [
      { key: "a", label: "", index: "", fields: [] },
      { key: "b", label: "", index: "", fields: [] },
    ];
    const { form } = await renderTree([GROUP], {
      groupKinds: [{ name: "data", create: async () => resolved }],
    });
    fireEvent.click(await screen.findByRole("button", { name: "Add data" }));
    await waitFor(() =>
      expect(
        form.current?.get<Item[]>("config.groups").value.map(({ key }) => key),
      ).toEqual(["g", "a", "b"]),
    );
    await screen.findByText("details-for-config.groups.a");
  });

  it("should replace a group whose key it holds", async () => {
    const replacement: Group = {
      ...GROUP,
      fields: [...GROUP.fields, field("w", "/wind")],
    };
    const { form } = await renderTree([GROUP], {
      groupKinds: [{ name: "data", create: async () => [replacement] }],
    });
    fireEvent.click(await screen.findByRole("button", { name: "Add data" }));
    await waitFor(() =>
      expect(form.current?.get<Item[]>("config.groups").value).toEqual([replacement]),
    );
  });

  it("should add a field copied from the last one, without its channel", async () => {
    const { form } = await renderTree([GROUP]);
    await screen.findAllByText("group-g");
    fireEvent.click(screen.getByRole("button", { name: "Add field" }));
    await waitFor(() =>
      expect(form.current?.get("config.groups.g.fields").value).toHaveLength(3),
    );
    const added = form.current?.get<Field[]>("config.groups.g.fields").value[2];
    expect(added?.pointer).toBe("/value");
    expect(added?.channel).toBe(0);
  });

  it("should duplicate a group with fresh fields and a remapped index", async () => {
    const { form } = await renderTree([GROUP]);
    // The selected group also names the details header; the tree row comes first.
    fireEvent.contextMenu((await screen.findAllByText("group-g"))[0]);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get("config.groups").value).toHaveLength(2),
    );
    const copy = form.current?.get<Group[]>("config.groups").value[1];
    expect(copy?.key).not.toBe("g");
    expect(copy?.fields.map((f) => f.channel)).toEqual([0, 0]);
    expect(copy?.index).toBe(copy?.fields[0].key);
    expect(copy?.index).not.toBe("ts");
  });

  it("should remove a field through the context menu", async () => {
    const { form } = await renderTree([GROUP]);
    fireEvent.contextMenu(await screen.findByText("/value"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() =>
      expect(form.current?.get("config.groups.g.fields").value).toHaveLength(1),
    );
  });

  it("should name a field with an empty pointer by the empty pointer label", async () => {
    await renderTree([{ ...GROUP, fields: [...GROUP.fields, field("e", "")] }], {
      emptyPointerLabel: "Whole payload",
    });
    await screen.findByText("Whole payload");
  });

  it("should add a group of the kind whose button is pressed", async () => {
    const { form } = await renderTree([]);
    fireEvent.click(await screen.findByRole("button", { name: "Add leaf" }));
    await waitFor(() =>
      expect(form.current?.get("config.groups").value).toHaveLength(1),
    );
    const [added] = form.current?.get<Leaf[]>("config.groups").value ?? [];
    expect(added).not.toHaveProperty("fields");
    await screen.findByText(`details-for-config.groups.${added.key}`);
  });

  it("should offer no add field action on a leaf group", async () => {
    await renderTree([LEAF]);
    const item = await screen.findByRole("treeitem", { name: /group-l/ });
    expect(within(item).queryByRole("button", { name: "Add field" })).toBeNull();
    fireEvent.contextMenu(item);
    await screen.findByText("Duplicate");
    expect(screen.queryByText("Add field")).toBeNull();
  });

  it("should duplicate a leaf group without its channels", async () => {
    const { form } = await renderTree([LEAF]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /group-l/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get("config.groups").value).toHaveLength(2),
    );
    const copy = form.current?.get<Leaf[]>("config.groups").value[1];
    expect(copy).toMatchObject({ channel: 0, name: "", index: 0 });
    expect(copy?.key).not.toBe("l");
  });

  it("should switch a group with a disabled flag through its checkbox and menu", async () => {
    const { form } = await renderTree([{ ...GROUP, disabled: false }]);
    const disabled = () => form.current?.get("config.groups.g.disabled").value;
    const item = await screen.findByRole("treeitem", { name: /group-g/ });
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(() => expect(disabled()).toBe(true));
    fireEvent.contextMenu(item);
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(() => expect(disabled()).toBe(false));
  });

  it("should offer no switch on a group without a disabled flag", async () => {
    await renderTree([GROUP]);
    const item = await screen.findByRole("treeitem", { name: /group-g/ });
    expect(within(item).queryByRole("checkbox")).toBeNull();
    fireEvent.contextMenu(item);
    await screen.findByText("Duplicate");
    expect(screen.queryByText("Disable")).toBeNull();
  });
});
