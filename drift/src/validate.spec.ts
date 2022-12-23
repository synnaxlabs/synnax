import { describe, expect, it } from "vitest";

import { validateAction } from "./validate";

describe("validate", () => {
  it("should throw an error if an action is undefined", () => {
    expect(() => validateAction({})).toThrowError();
  });
  it("should throw an error if an action type is undefined or an empty string", () => {
    expect(() => validateAction({ action: { type: "" } })).toThrowError();
    expect(() => validateAction({ action: { type: undefined } })).toThrowError();
  });
  it("should not throw an error for a valid action", () => {
    expect(() => validateAction({ action: { type: "type" } })).not.toThrowError();
  });
});
