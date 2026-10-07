// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type PageNavNode } from "@/components/nav/Page";
import { editionOf, groupOf, neighbors } from "@/components/nav/tree";

const NODES: PageNavNode[] = [
  { key: "/reference/", href: "/reference/", name: "Get started" },
  {
    key: "channels",
    name: "Channels",
    children: [
      { key: "/reference/channels", href: "/reference/channels", name: "Channels" },
      {
        key: "/reference/channels/create",
        href: "/reference/channels/create",
        name: "Create channels",
      },
    ],
  },
  {
    key: "enterprise",
    name: "Enterprise",
    edition: "enterprise",
    children: [
      { key: "/reference/enterprise", href: "/reference/enterprise", name: "Set up" },
      {
        key: "advanced",
        name: "Advanced",
        children: [
          {
            key: "/reference/enterprise/cli",
            href: "/reference/enterprise/cli",
            name: "CLI",
          },
        ],
      },
    ],
  },
  {
    key: "/reference/desktop",
    href: "/reference/desktop",
    name: "Desktop",
    edition: "desktop",
  },
];

describe("tree", () => {
  describe("neighbors", () => {
    it("should link across section boundaries in depth-first order", () => {
      const { prev, next } = neighbors(NODES, "/reference/channels/create");
      expect(prev?.href).toBe("/reference/channels");
      expect(next?.href).toBe("/reference/enterprise");
    });

    it("should skip nodes that have no page", () => {
      const { prev, next } = neighbors(NODES, "/reference/enterprise");
      expect(prev?.href).toBe("/reference/channels/create");
      expect(next?.href).toBe("/reference/enterprise/cli");
    });

    it("should leave the first page without a previous page", () => {
      const { prev, next } = neighbors(NODES, "/reference/");
      expect(prev).toBeUndefined();
      expect(next?.href).toBe("/reference/channels");
    });

    it("should leave the last page without a next page", () => {
      const { prev, next } = neighbors(NODES, "/reference/desktop");
      expect(prev?.href).toBe("/reference/enterprise/cli");
      expect(next).toBeUndefined();
    });

    it("should ignore a trailing slash on the path", () => {
      expect(neighbors(NODES, "/reference/channels/").next?.href).toBe(
        "/reference/channels/create",
      );
    });

    it("should return no neighbors for a path outside the tree", () => {
      expect(neighbors(NODES, "/reference/missing")).toEqual({});
    });
  });

  describe("editionOf", () => {
    it("should return undefined for a page that applies to every edition", () => {
      expect(editionOf(NODES, "/reference/channels/create")).toBeUndefined();
    });

    it("should inherit the edition of the nearest ancestor", () => {
      expect(editionOf(NODES, "/reference/enterprise/cli")).toBe("enterprise");
    });

    it("should read the edition set on the page itself", () => {
      expect(editionOf(NODES, "/reference/desktop")).toBe("desktop");
    });

    it("should return undefined for a path outside the tree", () => {
      expect(editionOf(NODES, "/reference/missing")).toBeUndefined();
    });
  });

  describe("groupOf", () => {
    it("should return the group that holds the page", () => {
      expect(groupOf(NODES, "/reference/channels/create")).toBe("Channels");
    });

    it("should return the nearest group of a nested page", () => {
      expect(groupOf(NODES, "/reference/enterprise/cli")).toBe("Advanced");
    });

    it("should return undefined for a page outside every group", () => {
      expect(groupOf(NODES, "/reference/desktop")).toBeUndefined();
    });

    it("should return undefined for a path outside the tree", () => {
      expect(groupOf(NODES, "/reference/missing")).toBeUndefined();
    });
  });
});
