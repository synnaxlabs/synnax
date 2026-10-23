// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { QUERY } from "@/components/client/choice";
import { Var } from "@/components/client/Var";

describe("client choice", () => {
  it("should show the TypeScript form of inline values for TypeScript", () => {
    document.body.innerHTML = renderToString(
      <Var py="set_authority" ts="setAuthority" />,
    );
    QUERY.onChange?.("typescript");
    expect(document.body.textContent).toBe("setAuthority");
  });

  it("should show the Python form of inline values for any other client", () => {
    document.body.innerHTML = renderToString(
      <Var py="set_authority" ts="setAuthority" />,
    );
    QUERY.onChange?.("typescript");
    QUERY.onChange?.("console");
    expect(document.body.textContent).toBe("set_authority");
  });
});
