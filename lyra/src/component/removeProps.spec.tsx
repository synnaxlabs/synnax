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

import { removeProps } from "@/component/removeProps";

describe("removeProps", () => {
  // Test component that we'll use
  const TestComponent = ({
    unwantedProp,
    anotherProp,
  }: {
    unwantedProp?: string;
    anotherProp?: string;
  }) => {
    if (unwantedProp != null) return <div>{unwantedProp}</div>;
    return <div>{anotherProp}</div>;
  };
  TestComponent.displayName = "TestComponent";

  it("should remove specified props from the component", () => {
    const WrappedComponent = removeProps(TestComponent, ["unwantedProp"]);
    const c = render(
      <WrappedComponent unwantedProp="should not pass" anotherProp="should pass" />,
    );

    expect(c.queryByText("should pass")).not.toBeNull();
    expect(c.queryByText("should not pass")).toBeNull();
  });
});
