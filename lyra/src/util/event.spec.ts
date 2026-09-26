// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { blockActivation, isInputOrContentEditable } from "@/util/event";

const event = (key: string) => {
  let prevented = false;
  return {
    e: { key, preventDefault: () => (prevented = true) },
    prevented: () => prevented,
  };
};

describe("blockActivation", () => {
  it.each([" ", "Enter"])("should prevent the default for %j", (key) => {
    const { e, prevented } = event(key);
    blockActivation(e);
    expect(prevented()).toBe(true);
  });

  it("should leave other keys untouched", () => {
    const { e, prevented } = event("a");
    blockActivation(e);
    expect(prevented()).toBe(false);
  });
});

describe("isInputOrContentEditable", () => {
  it.each([
    ["an input", document.createElement("input")],
    ["a textarea", document.createElement("textarea")],
  ])("should report %s as text entry", (_, target) => {
    expect(isInputOrContentEditable({ target })).toBe(true);
  });

  it("should report a contenteditable element as text entry", () => {
    const target = document.createElement("div");
    target.setAttribute("contenteditable", "true");
    expect(isInputOrContentEditable({ target })).toBe(true);
  });

  it("should report a textbox role as text entry", () => {
    const target = document.createElement("div");
    target.role = "textbox";
    expect(isInputOrContentEditable({ target })).toBe(true);
  });

  it.each([
    ["a button", document.createElement("button")],
    ["a plain div", document.createElement("div")],
  ])("should not report %s as text entry", (_, target) => {
    expect(isInputOrContentEditable({ target })).toBe(false);
  });

  it("should not report a missing target as text entry", () => {
    expect(isInputOrContentEditable({ target: null })).toBe(false);
  });
});
