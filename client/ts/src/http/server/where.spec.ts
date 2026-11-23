// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { http } from "@/http";
import { matches, serverComputed } from "@/http/server/where.gen";
import { vectors } from "@/testutil/vectors";

describe("Where", () => {
  const { records, cases, whereZ } = vectors.read(
    "httpServers",
    http.server.serverZ,
    http.server.whereZ,
  );
  it.each(cases.map((c) => [c.name, c] as const))("should decide %s", (_, c) => {
    const where = whereZ.parse(c.where);
    expect(serverComputed(where)).toBe(c.serverComputed ?? false);
    if (c.serverComputed === true) return;
    const matched = records
      .filter(([, r]) => matches(r, where))
      .map(([alias]) => alias);
    expect(matched.sort()).toEqual([...c.expected].sort());
  });
});
