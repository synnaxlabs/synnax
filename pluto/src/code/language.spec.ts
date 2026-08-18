// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { readFileSync } from "node:fs";
import { createRequire } from "node:module";

import { beforeEach, describe, expect, it, vi } from "vitest";

import { BASE_THEMES, type Language, registerLanguage } from "@/code/language";

const { registerExtensionMock, registerFileURLMock, updateMock } = vi.hoisted(() => ({
  registerExtensionMock: vi.fn(),
  registerFileURLMock: vi.fn(),
  updateMock: vi.fn(),
}));
vi.mock("@codingame/monaco-vscode-api/extensions", () => ({
  registerExtension: registerExtensionMock,
  ExtensionHostKind: { LocalProcess: 1 },
}));
vi.mock("@codingame/monaco-vscode-extension-api", () => ({
  workspace: { getConfiguration: () => ({ update: updateMock }) },
  ConfigurationTarget: { Global: 1 },
}));

interface ThemeManifest {
  contributes: { themes: { id: string }[] };
}

const readThemeManifest = (): ThemeManifest => {
  const path = createRequire(import.meta.url).resolve(
    "@codingame/monaco-vscode-theme-defaults-default-extension/resources/package.json",
  );
  return JSON.parse(readFileSync(path, "utf-8")) as ThemeManifest;
};

const LANGUAGE: Language = {
  name: "arc",
  configuration: "{}",
  grammar: { scopeName: "source.arc", raw: "{}" },
  theme: {
    semanticTokenColors: { dark: { keyword: "#fff" }, light: { keyword: "#000" } },
    textMateRules: {
      dark: [{ scope: "keyword.arc", settings: { foreground: "#fff" } }],
      light: [{ scope: "keyword.arc", settings: { foreground: "#000" } }],
    },
  },
};

describe("language", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    registerExtensionMock.mockReturnValue({ registerFileUrl: registerFileURLMock });
  });

  describe("BASE_THEMES", () => {
    // VS Code renamed these ids (dropping a "Default " prefix) in the upgrade to
    // monaco-vscode-api v36. Nothing failed loudly: the base theme still resolved
    // through a compatibility shim, but the token color customizations are matched
    // literally against the theme's settings id, so every language color was dropped.
    it("should name themes the installed VS Code theme extension contributes", () => {
      const ids = readThemeManifest().contributes.themes.map(({ id }) => id);
      Object.values(BASE_THEMES).forEach((theme) => expect(ids).toContain(theme));
    });
  });

  describe("registerLanguage", () => {
    it("should scope the token colors to the base themes the editor selects", async () => {
      await registerLanguage(LANGUAGE);
      const scopes = updateMock.mock.calls.map(([, value]) => Object.keys(value));
      expect(scopes).toEqual([
        [`[${BASE_THEMES.dark}]`, `[${BASE_THEMES.light}]`],
        [`[${BASE_THEMES.dark}]`, `[${BASE_THEMES.light}]`],
      ]);
    });

    it("should register the grammar and configuration files", async () => {
      await registerLanguage(LANGUAGE);
      const paths = registerFileURLMock.mock.calls.map(([path]) => path);
      expect(paths).toEqual([
        "./arc.tmLanguage.json",
        "./arc.language-configuration.json",
      ]);
    });
  });
});
