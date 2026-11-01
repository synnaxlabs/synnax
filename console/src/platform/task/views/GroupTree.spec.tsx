// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { deep } from "@synnaxlabs/x";
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

const path = deep.path<{ config: { groups: Group[] } }>();

const groupLabel = Component.renderProp(
  ({ groupKey }: Task.Views.GroupProps<Group>) => <span>group-{groupKey}</span>,
);

const groupDetails = Component.renderProp(({ path }: Task.Views.GroupProps<Group>) => (
  <span>details-for-{String(path)}</span>
));

const fieldDetails = Component.renderProp(({ path }: Task.Views.FieldProps<Field>) => (
  <span>details-for-{String(path)}</span>
));

const createGroup = (): Omit<Group, "key"> => ({ label: "", index: "", fields: [] });

const createField = (): Omit<Field, "key"> => field("", "");

const renderTree = (groups: Group[], mode?: "preview") =>
  renderInTaskForm(
    <Task.Views.GroupTree<Group, Field>
      path={path.config.groups}
      listTitle="Groups"
      groupName="group"
      placeholder="Select a group or field"
      createGroup={createGroup}
      createField={createField}
      groupLabel={groupLabel}
      groupDetails={groupDetails}
      fieldDetails={fieldDetails}
    />,
    { values: { config: { groups } }, mode },
  );

const GROUP: Group = {
  key: "g",
  label: "",
  index: "ts",
  fields: [field("ts", "/time", 1), field("v", "/value", 2)],
};

const EMPTY_GROUP: Group = { key: "e", label: "", index: "", fields: [] };

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
      expect(form.current?.get(path.config.groups).value).toHaveLength(1),
    );
    const [added] = form.current?.get(path.config.groups).value ?? [];
    await screen.findByText(`details-for-config.groups.${added.key}`);
  });

  it("should add a field copied from the last one, without its channel", async () => {
    const { form } = await renderTree([GROUP]);
    await screen.findAllByText("group-g");
    fireEvent.click(screen.getByRole("button", { name: "Add field" }));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups.g.fields).value).toHaveLength(3),
    );
    const added = form.current?.get(path.config.groups.g.fields).value[2];
    expect(added?.pointer).toBe("/value");
    expect(added?.channel).toBe(0);
  });

  it("should offer an add field item only under a group with no fields", async () => {
    await renderTree([EMPTY_GROUP, GROUP]);
    await screen.findAllByText("group-e");
    expect(screen.getAllByText("Add field")).toHaveLength(1);
  });

  it("should add and select the first field when the add field item is chosen", async () => {
    const { form } = await renderTree([EMPTY_GROUP]);
    fireEvent.click(await screen.findByText("Add field"));
    const { fields } = path.config.groups.e;
    await waitFor(() => expect(form.current?.get(fields).value).toHaveLength(1));
    const [added] = form.current?.get(fields).value ?? [];
    await screen.findByText(`details-for-config.groups.e.fields.${added.key}`);
    expect(screen.queryByText("Add field")).toBeNull();
  });

  it("should not offer an add field item in preview", async () => {
    await renderTree([EMPTY_GROUP], "preview");
    await screen.findAllByText("group-e");
    expect(screen.queryByText("Add field")).toBeNull();
  });

  it("should not offer item actions on an add field item", async () => {
    await renderTree([EMPTY_GROUP]);
    fireEvent.contextMenu(await screen.findByText("Add field"));
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Remove")).toBeNull();
  });

  it("should not move the selection onto an add field item after a remove", async () => {
    await renderTree([EMPTY_GROUP, GROUP]);
    fireEvent.click((await screen.findAllByText("group-g"))[0]);
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.click(screen.getByText("/value"));
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
    fireEvent.contextMenu(screen.getAllByText("/value")[0]);
    fireEvent.click(await screen.findByText("Remove"));
    await screen.findByText("details-for-config.groups.e");
  });

  it("should duplicate a group with fresh fields and a remapped index", async () => {
    const { form } = await renderTree([GROUP]);
    // The selected group also names the details header; the tree row comes first.
    fireEvent.contextMenu((await screen.findAllByText("group-g"))[0]);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups).value).toHaveLength(2),
    );
    const copy = form.current?.get(path.config.groups).value[1];
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
      expect(form.current?.get(path.config.groups.g.fields).value).toHaveLength(1),
    );
  });
});
