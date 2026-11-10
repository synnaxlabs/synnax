// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Nav } from "@/components/nav";

const NODES: Nav.PageNavNode[] = [
  { href: "/reference/", name: "Get started" },
  {
    key: "channels",
    name: "Channels",
    children: [
      { href: "/reference/channels", name: "Channels" },
      {
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
      { href: "/reference/enterprise", name: "Set up" },
      {
        key: "advanced",
        name: "Advanced",
        children: [
          {
            href: "/reference/enterprise/cli",
            name: "CLI",
          },
        ],
      },
    ],
  },
  {
    href: "/reference/desktop",
    name: "Desktop",
    edition: "desktop",
  },
];

describe("tree", () => {
  describe("neighbors", () => {
    it("should link across section boundaries in depth-first order", () => {
      const { prev, next } = Nav.neighbors(NODES, "/reference/channels/create");
      expect(prev?.href).toBe("/reference/channels");
      expect(next?.href).toBe("/reference/enterprise");
    });

    it("should skip nodes that have no page", () => {
      const { prev, next } = Nav.neighbors(NODES, "/reference/enterprise");
      expect(prev?.href).toBe("/reference/channels/create");
      expect(next?.href).toBe("/reference/enterprise/cli");
    });

    it("should leave the first page without a previous page", () => {
      const { prev, next } = Nav.neighbors(NODES, "/reference/");
      expect(prev).toBeUndefined();
      expect(next?.href).toBe("/reference/channels");
    });

    it("should leave the last page without a next page", () => {
      const { prev, next } = Nav.neighbors(NODES, "/reference/desktop");
      expect(prev?.href).toBe("/reference/enterprise/cli");
      expect(next).toBeUndefined();
    });

    it("should ignore a trailing slash on the path", () => {
      expect(Nav.neighbors(NODES, "/reference/channels/").next?.href).toBe(
        "/reference/channels/create",
      );
    });

    it("should return no neighbors for a path outside the tree", () => {
      expect(Nav.neighbors(NODES, "/reference/missing")).toEqual({});
    });
  });

  describe("editionOf", () => {
    it("should return undefined for a page that applies to every edition", () => {
      expect(Nav.editionOf(NODES, "/reference/channels/create")).toBeUndefined();
    });

    it("should inherit the edition of the nearest ancestor", () => {
      expect(Nav.editionOf(NODES, "/reference/enterprise/cli")).toBe("enterprise");
    });

    it("should read the edition set on the page itself", () => {
      expect(Nav.editionOf(NODES, "/reference/desktop")).toBe("desktop");
    });

    it("should return undefined for a path outside the tree", () => {
      expect(Nav.editionOf(NODES, "/reference/missing")).toBeUndefined();
    });
  });

  describe("groupOf", () => {
    it("should return the group that holds the page", () => {
      expect(Nav.groupOf(NODES, "/reference/channels/create")).toBe("Channels");
    });

    it("should return the nearest group of a nested page", () => {
      expect(Nav.groupOf(NODES, "/reference/enterprise/cli")).toBe("Advanced");
    });

    it("should return undefined for a page outside every group", () => {
      expect(Nav.groupOf(NODES, "/reference/desktop")).toBeUndefined();
    });

    it("should return undefined for a path outside the tree", () => {
      expect(Nav.groupOf(NODES, "/reference/missing")).toBeUndefined();
    });
  });

  describe("sectionOf", () => {
    it("should return the top-level group that holds the page", () => {
      expect(Nav.sectionOf(NODES, "/reference/enterprise/cli")).toBe("Enterprise");
    });

    it("should return undefined for a top-level page", () => {
      expect(Nav.sectionOf(NODES, "/reference/desktop")).toBeUndefined();
    });

    it("should return undefined for a path outside the tree", () => {
      expect(Nav.sectionOf(NODES, "/reference/missing")).toBeUndefined();
    });
  });

  describe("firstPages", () => {
    const GROUPS: Nav.PageNavNode[] = [
      {
        key: "get-started",
        name: "Get started",
        children: [
          { href: "/reference/", name: "Overview" },
          { href: "/reference/get-started/concepts", name: "Concepts" },
        ],
      },
      {
        key: "hardware",
        name: "Hardware",
        children: [
          { href: "/reference/hardware", name: "Overview" },
          {
            key: "ni",
            name: "NI",
            children: [
              {
                key: "devices",
                name: "Devices",
                children: [{ href: "/reference/hardware/ni/devices/add", name: "Add" }],
              },
            ],
          },
        ],
      },
      {
        href: "/reference/arc",
        name: "Arc",
        children: [
          {
            key: "library",
            name: "Library",
            children: [{ href: "/reference/arc/library/math", name: "math" }],
          },
        ],
      },
    ];
    const pages = Nav.firstPages(GROUPS, "/reference");

    it("should map a group to its first page as the nav writes it", () => {
      expect(pages.get("/reference/get-started")).toBe("/reference/");
    });

    it("should skip a group whose route is already a page", () => {
      expect(pages.has("/reference/hardware")).toBe(false);
    });

    it("should map a group whose first page sits in a nested group", () => {
      expect(pages.get("/reference/hardware/ni")).toBe(
        "/reference/hardware/ni/devices/add",
      );
      expect(pages.get("/reference/hardware/ni/devices")).toBe(
        "/reference/hardware/ni/devices/add",
      );
    });

    it("should place a group under a page at the route of the page", () => {
      expect(pages.get("/reference/arc/library")).toBe("/reference/arc/library/math");
    });

    it("should map nothing but group routes", () => {
      expect([...pages.keys()].sort()).toEqual([
        "/reference/arc/library",
        "/reference/get-started",
        "/reference/hardware/ni",
        "/reference/hardware/ni/devices",
      ]);
    });
  });

  describe("labels", () => {
    it("should name a page after its node", () => {
      expect(Nav.labels(NODES).get("/reference/channels/create")).toBe(
        "Create channels",
      );
    });

    it("should name the route that a group's pages share after the group", () => {
      expect(Nav.labels(NODES).get("/reference/enterprise")).toBe("Enterprise");
    });

    it("should name a route that nested groups share after the outermost one", () => {
      const named = Nav.labels([
        {
          key: "enterprise",
          name: "Enterprise",
          children: [
            { href: "/reference/enterprise", name: "Set up" },
            {
              key: "tls",
              name: "TLS",
              children: [
                { href: "/reference/enterprise/tls", name: "Set up TLS" },
                { href: "/reference/enterprise/certificates", name: "Certificates" },
              ],
            },
          ],
        },
        { href: "/reference/desktop", name: "Desktop" },
      ]);
      expect(named.get("/reference/enterprise")).toBe("Enterprise");
      expect(named.get("/reference/enterprise/tls")).toBe("Set up TLS");
    });

    it("should leave the route that every page shares without a name", () => {
      expect(Nav.labels(NODES).has("/reference")).toBe(false);
    });
  });
});
