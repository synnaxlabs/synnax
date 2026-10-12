// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { fireEvent, screen, waitFor } from "@testing-library/react";
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

const createGroup = (): Omit<Group, "key"> => ({ label: "", index: "", fields: [] });

const createField = (): Omit<Field, "key"> => field("", "");

const renderTree = (groups: Group[]) =>
  renderInTaskForm(
    <Task.Views.GroupTree<Group, Field>
      listTitle="Groups"
      groupName="group"
      placeholder="Select a group or field"
      createGroup={createGroup}
      createField={createField}
      groupLabel={groupLabel}
      groupDetails={groupDetails}
      fieldDetails={fieldDetails}
    />,
    { values: { config: { groups } } },
  );

const GROUP: Group = {
  key: "g",
  label: "",
  index: "ts",
  fields: [field("ts", "/time", 1), field("v", "/value", 2)],
};

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
});
