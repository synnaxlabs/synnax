// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// <reference types="vitest/config" />

import { lib } from "@synnaxlabs/vite-plugin";
import path from "path";
import { defineConfig } from "vite";

const r = (p: string) => path.resolve(".", p);

export default defineConfig({
  base: "/x/",
  plugins: [lib({ name: "x" })],
  build: {
    lib: {
      entry: {
        array: r("src/array/index.ts"),
        binary: r("src/binary/index.ts"),
        bounds: r("src/spatial/bounds/index.ts"),
        box: r("src/spatial/box/index.ts"),
        breaker: r("src/breaker/index.ts"),
        caseconv: r("src/caseconv/index.ts"),
        change: r("src/change/index.ts"),
        clamp: r("src/clamp/index.ts"),
        color: r("src/color/index.ts"),
        compare: r("src/compare/index.ts"),
        control: r("src/control/index.ts"),
        csv: r("src/csv/index.ts"),
        debounce: r("src/debounce/index.ts"),
        deep: r("src/deep/index.ts"),
        destructor: r("src/destructor/index.ts"),
        dimensions: r("src/spatial/dimensions/index.ts"),
        direction: r("src/spatial/direction/index.ts"),
        errors: r("src/errors/index.ts"),
        fmt: r("src/fmt/index.ts"),
        id: r("src/id/index.ts"),
        instance: r("src/instance/index.ts"),
        json: r("src/json/index.ts"),
        kv: r("src/kv/index.ts"),
        label: r("src/label/index.ts"),
        link: r("src/link/index.ts"),
        location: r("src/spatial/location/index.ts"),
        map: r("src/map/index.ts"),
        math: r("src/math/index.ts"),
        migrate: r("src/migrate/index.ts"),
        narrow: r("src/narrow/index.ts"),
        notation: r("src/notation/index.ts"),
        numeric: r("src/numeric/index.ts"),
        observe: r("src/observe/index.ts"),
        optional: r("src/optional/index.ts"),
        primitive: r("src/primitive/index.ts"),
        record: r("src/record/index.ts"),
        require: r("src/require/index.ts"),
        runtime: r("src/runtime/index.ts"),
        scale: r("src/spatial/scale/index.ts"),
        scheduler: r("src/scheduler/index.ts"),
        shallow: r("src/shallow/index.ts"),
        singleton: r("src/singleton/index.ts"),
        sleep: r("src/sleep/index.ts"),
        spatial: r("src/spatial/spatial.ts"),
        status: r("src/status/index.ts"),
        sticky: r("src/spatial/sticky/index.ts"),
        strings: r("src/strings/index.ts"),
        sync: r("src/sync/index.ts"),
        telem: r("src/telem/index.ts"),
        testutil: r("src/testutil/index.ts"),
        throttle: r("src/throttle/index.ts"),
        types: r("src/types/index.ts"),
        unique: r("src/unique/index.ts"),
        url: r("src/url/index.ts"),
        uuid: r("src/uuid/index.ts"),
        xy: r("src/spatial/xy/index.ts"),
        zod: r("src/zod/index.ts"),
      },
    },
    rolldownOptions: {
      external: ["async-mutex", "nanoid", "uuid", "zod"],
      output: { preserveModules: false },
    },
  },
  test: {
    exclude: ["**/node_modules/**", "**/dist/**"],
    coverage: {
      include: ["src/**/*.ts", "src/**/*.tsx"],
      exclude: ["src/**/*.spec.ts", "src/**/*.spec.tsx", "src/**/*.bench.ts"],
    },
  },
});
