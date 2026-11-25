// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, onTestFinished } from "vitest";

import { Core } from "@/platform/core";
import { Session } from "@/session";

const ADDRESS = { host: "core.example.com", port: 9090, secure: true };
const CODE = "syi_abc-DEF_123";

const open = (url: string): void => {
  window.history.replaceState(null, "", url);
  onTestFinished(() => window.history.replaceState(null, "", "/"));
};

describe("invite links", () => {
  it("should put the code in the fragment of the Core's own page", () => {
    const { browser } = Core.createInviteLinks(ADDRESS, CODE);
    expect(browser).toBe(`https://core.example.com:9090/#invite=${CODE}`);
    const url = new URL(browser);
    expect(url.pathname + url.search).not.toContain(CODE);
    open(`/${url.hash}`);
    expect(Core.takeBrowserInvite()?.code).toBe(CODE);
  });

  it("should put an IPv6 host in brackets", () => {
    const { browser } = Core.createInviteLinks({ ...ADDRESS, host: "::1" }, CODE);
    expect(new URL(browser).hostname).toBe("[::1]");
  });

  it("should use http for a Core that is not secure", () => {
    const { browser } = Core.createInviteLinks({ ...ADDRESS, secure: false }, CODE);
    expect(browser.startsWith("http://core.example.com:9090/#")).toBe(true);
  });

  it("should carry the Core's address and the code in a desktop link", () => {
    const { desktop } = Core.createInviteLinks(ADDRESS, CODE);
    expect(desktop.startsWith(Core.DESKTOP_INVITE_PREFIX)).toBe(true);
    expect(Core.parseDesktopInvite(desktop)).toEqual({ ...ADDRESS, code: CODE });
  });

  it("should read an insecure Core from a desktop link", () => {
    const { desktop } = Core.createInviteLinks({ ...ADDRESS, secure: false }, CODE);
    expect(Core.parseDesktopInvite(desktop).secure).toBe(false);
  });

  it("should take no code from a fragment with a blank one", () => {
    open("/#invite=%20");
    expect(Core.takeBrowserInvite()).toBeNull();
  });

  it("should refuse a desktop link without a code", () => {
    expect(() =>
      Core.parseDesktopInvite("synnax://invite?host=core.example.com&port=9090"),
    ).toThrow("The invite link has no invite code.");
  });

  it("should refuse a desktop link without an address", () => {
    expect(() =>
      Core.parseDesktopInvite(`synnax://invite?port=9090#invite=${CODE}`),
    ).toThrow("The invite link has no Core address.");
    expect(() =>
      Core.parseDesktopInvite(`synnax://invite?host=a&port=x#invite=${CODE}`),
    ).toThrow("The invite link has no Core address.");
  });

  describe("takeBrowserInvite", () => {
    it("should take the code for the serving Core and clear the fragment", () => {
      open(`/?keep=1#invite=${CODE}`);
      expect(Core.takeBrowserInvite()).toEqual({
        core: Session.Core.SERVED_KEY,
        code: CODE,
      });
      expect(window.location.hash).toBe("");
      expect(window.location.search).toBe("?keep=1");
    });

    it("should leave an address without a code alone", () => {
      open("/?keep=1#other=1");
      expect(Core.takeBrowserInvite()).toBeNull();
      expect(window.location.hash).toBe("#other=1");
    });
  });
});
