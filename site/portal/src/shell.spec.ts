// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type Organization } from "@/server/db/schema";
import { pick } from "@/server/organization";
import { HOME, landing, scoped, tabs } from "@/shell";

const organization = (key: string, kind: Organization["kind"]): Organization => ({
  key,
  kind,
  name: key,
  clerkOrgID: kind === "team" ? `clerk-${key}` : null,
  ownerUserID: kind === "personal" ? "user" : null,
  createdAt: new Date(0),
});

const PERSONAL = organization("personal", "personal");
const ACME = organization("acme", "team");
const GLOBEX = organization("globex", "team");

describe("shell", () => {
  describe("landing", () => {
    const land = (search: string): string => landing(new URLSearchParams(search));
    it("should land on the page that sent the visitor", () => {
      expect(land("redirect_url=%2Fdesktop%2Flogin%3Fstate%3Dabc")).toBe(
        "/desktop/login?state=abc",
      );
    });
    it("should land home without a redirect", () => {
      expect(land("")).toBe(HOME);
    });
    it("should land home for another site", () => {
      expect(land("redirect_url=https%3A%2F%2Fevil.example")).toBe(HOME);
    });
    it("should land home for a protocol-relative path", () => {
      expect(land("redirect_url=%2F%2Fevil.example")).toBe(HOME);
    });
  });
  describe("tabs", () => {
    it("should give a personal scope its overview and devices", () => {
      expect(tabs(PERSONAL, false)).toEqual([
        { tab: "overview", label: "Overview", href: "/?org=personal" },
        { tab: "devices", label: "Devices", href: "/devices?org=personal" },
      ]);
    });
    it("should give a team scope its overview, licenses, and members", () => {
      expect(tabs(ACME, false)).toEqual([
        { tab: "overview", label: "Overview", href: "/?org=acme" },
        { tab: "licenses", label: "Licenses", href: "/licenses?org=acme" },
        { tab: "members", label: "Members", href: "/members?org=acme" },
      ]);
    });
    it("should add an unscoped admin tab for staff", () => {
      expect(tabs(PERSONAL, true).at(-1)).toEqual({
        tab: "admin",
        label: "Admin",
        href: "/admin",
      });
    });
  });
  describe("scoped", () => {
    it("should add the scope to a path", () => {
      expect(scoped("/licenses/activate", ACME)).toBe("/licenses/activate?org=acme");
    });
  });
  describe("pick", () => {
    const all = [PERSONAL, ACME, GLOBEX];
    it("should default to the first team", () => {
      expect(pick(all, null, null)).toBe(ACME);
    });
    it("should default to the personal organization without a team", () => {
      expect(pick([PERSONAL], null, null)).toBe(PERSONAL);
    });
    it("should select the personal organization by key", () => {
      expect(pick(all, "personal", null)).toBe(PERSONAL);
    });
    it("should select a team by key", () => {
      expect(pick(all, "globex", null)).toBe(GLOBEX);
    });
    it("should return null for a key that is not the user's", () => {
      expect(pick(all, "initech", null)).toBeNull();
    });
    it("should act for the remembered scope without a request", () => {
      expect(pick(all, null, "personal")).toBe(PERSONAL);
    });
    it("should prefer the requested scope over the remembered one", () => {
      expect(pick(all, "globex", "personal")).toBe(GLOBEX);
    });
    it("should fall back to the default when the remembered scope is gone", () => {
      expect(pick(all, null, "initech")).toBe(ACME);
    });
  });
});
