// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Generic } from "@/generic";

describe("Generic", () => {
  it("should render a div", () => {
    const { container } = render(<Generic.Element el="input" />);
    expect(container.firstChild).toBeInstanceOf(HTMLInputElement);
  });
});
