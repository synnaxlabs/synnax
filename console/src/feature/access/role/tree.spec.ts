// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, createTestClient } from "@synnaxlabs/client";
import { type Haul, User } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Access } from "@/feature/access";
import {
  findModalButton,
  renderTreeContextMenu,
} from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import { assertDefined, uniqueName } from "@/testutil";

const client = createTestClient();

const createRole = async () =>
  await client.access.roles.create({ name: uniqueName("role") });

const roleResource = (key: string, name: string, internal: boolean) =>
  createResource(access.role.ontologyID(key), name, { internal });

describe("role ontology service", () => {
  it("should expose rename and delete for a non-internal role", async () => {
    const role = await createRole();
    assertDefined(Access.Role.TREE_ITEM.ContextMenu);
    await renderTreeContextMenu(Access.Role.TREE_ITEM.ContextMenu, {
      client,
      resources: [roleResource(role.key, role.name, false)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should hide rename and delete for internal roles", async () => {
    const role = await createRole();
    assertDefined(Access.Role.TREE_ITEM.ContextMenu);
    await renderTreeContextMenu(Access.Role.TREE_ITEM.ContextMenu, {
      client,
      resources: [roleResource(role.key, role.name, true)],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  it("should delete the role on the cluster after confirmation", async () => {
    const role = await createRole();
    assertDefined(Access.Role.TREE_ITEM.ContextMenu);
    await renderTreeContextMenu(Access.Role.TREE_ITEM.ContextMenu, {
      client,
      resources: [roleResource(role.key, role.name, false)],
    });
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${role.name}?`);
    fireEvent.click(findModalButton("Delete"));
    const roleExists = async (): Promise<boolean> => {
      try {
        await client.access.roles.retrieve({ key: role.key });
        return true;
      } catch {
        return false;
      }
    };
    await waitFor(async () => expect(await roleExists()).toBe(false));
  });

  it("should accept only non-root user haul items on drop", () => {
    const { canDrop } = Access.Role.TREE_ITEM;
    const source: Haul.Item = { key: "tree", type: "Tree.Item" };
    const userItem = (rootUser: boolean): Haul.Item =>
      User.createHaulItem({
        key: "u1",
        username: "u",
        firstName: "",
        lastName: "",
        rootUser,
      });
    expect(canDrop({ source, items: [userItem(false)] })).toBe(true);
    expect(canDrop({ source, items: [userItem(true)] })).toBe(false);
    expect(canDrop({ source, items: [{ key: "channel:1", type: "channel" }] })).toBe(
      false,
    );
  });
});

describe("policy ontology service", () => {
  it("should stay hidden in the tree and declare no children", () => {
    expect(Access.Policy.TREE_ITEM.type).toBe("policy");
    expect(
      Access.Policy.TREE_ITEM.visible?.(
        createResource(access.policy.ontologyID("p1"), "p1"),
      ),
    ).toBe(false);
    expect(Access.Policy.TREE_ITEM.hasChildren).toBe(false);
  });
});
