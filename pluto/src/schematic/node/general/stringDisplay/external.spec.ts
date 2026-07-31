// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { GROUP } from "@/schematic/node/general/group";
import { StringDisplay } from "@/schematic/node/general/stringDisplay";
import { telem } from "@/telem/aether";
import { Theming } from "@/theming";

const theme = Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxDark);

describe("StringDisplay", () => {
  describe("defaultConfig", () => {
    it("should produce a config that satisfies its own schema", () => {
      const config = StringDisplay.defaultConfig(theme);
      expect(StringDisplay.configZ.parse(config)).toEqual(config);
    });

    it("should source telemetry from a bare string source, not a pipeline", () => {
      const { telem: t } = StringDisplay.defaultConfig(theme);
      expect(t?.variant).toBe("source");
      expect(t?.valueType).toBe("string");
      expect(t?.type).toBe("stream-channel-string-value");
    });
  });

  describe("configZ", () => {
    it("should reject a telem spec that emits numbers", () => {
      const config = {
        ...StringDisplay.defaultConfig(theme),
        telem: telem.streamChannelValue({ channel: 1 }),
      };
      expect(() => StringDisplay.configZ.parse(config)).toThrow();
    });
  });

  // Console's StaticSymbolList filters by group, so a symbol missing from
  // GROUP.symbols is reachable only through search.
  it("should be listed in the general group", () => {
    expect(GROUP.symbols).toContain("stringDisplay");
  });
});
