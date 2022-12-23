import { describe, expect, test } from "vitest";

import URL from "./url";

describe("URL", () => {
  test("URL - child", () => {
    const endpoint = new URL({
      host: "localhost",
      port: 8080,
      protocol: "http",
      pathPrefix: "api",
    });
    expect(endpoint.child("test").toString()).toEqual("http://localhost:8080/api/test");
  });

  test("URL - child with trailing slash", () => {
    const endpoint = new URL({
      host: "localhost",
      port: 8080,
      protocol: "http",
      pathPrefix: "api",
    });
    const child = endpoint.child("test/");
    expect(child.toString()).toEqual("http://localhost:8080/api/test");
  });

  test("URL - replacing protocol", () => {
    const endpoint = new URL({
      host: "localhost",
      port: 8080,
      protocol: "http",
      pathPrefix: "api",
    });
    expect(endpoint.child("test").replace({ protocol: "https" }).toString()).toEqual(
      "https://localhost:8080/api/test"
    );
  });
});
