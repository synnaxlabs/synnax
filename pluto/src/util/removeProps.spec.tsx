import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { removeProps } from "@/util/removeProps";

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
