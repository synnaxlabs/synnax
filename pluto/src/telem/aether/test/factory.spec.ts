// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, beforeEach, afterEach } from "vitest";

import { type Telem } from "@/telem/aether/telem";
import {
  registerTestInstance,
  TestFactory,
  unregisterTestInstance,
} from "@/telem/aether/test/factory";

const createMockTelem = (): Telem => ({
  cleanup: () => {},
});

describe("TestFactory", () => {
  describe("registerTestInstance and unregisterTestInstance", () => {
    it("should register an instance that can be retrieved by factory", () => {
      const instance = createMockTelem();
      const id = "test-id-1";

      registerTestInstance(id, instance);

      const factory = new TestFactory();
      const result = factory.create({ type: "any", props: { testId: id } });

      expect(result).toBe(instance);

      unregisterTestInstance(id);
    });

    it("should unregister an instance so factory returns null", () => {
      const instance = createMockTelem();
      const id = "test-id-2";

      registerTestInstance(id, instance);
      unregisterTestInstance(id);

      const factory = new TestFactory();
      const result = factory.create({ type: "any", props: { testId: id } });

      expect(result).toBeNull();
    });
  });

  describe("create", () => {
    let factory: TestFactory;
    let registeredIds: string[] = [];

    beforeEach(() => {
      factory = new TestFactory();
      registeredIds = [];
    });

    afterEach(() => {
      registeredIds.forEach((id) => unregisterTestInstance(id));
    });

    const registerInstance = (id: string, instance: Telem): void => {
      registerTestInstance(id, instance);
      registeredIds.push(id);
    };

    it("should return null for spec without testId", () => {
      const result = factory.create({ type: "test-sink", props: {} });
      expect(result).toBeNull();
    });

    it("should return null for spec without props", () => {
      const result = factory.create({ type: "test-sink" });
      expect(result).toBeNull();
    });

    it("should return null for unregistered testId", () => {
      const result = factory.create({
        type: "test-sink",
        props: { testId: "non-existent" },
      });
      expect(result).toBeNull();
    });

    it("should return registered instance for valid testId", () => {
      const instance = createMockTelem();
      registerInstance("valid-id", instance);

      const result = factory.create({
        type: "test-sink",
        props: { testId: "valid-id" },
      });

      expect(result).toBe(instance);
    });

    it("should handle multiple instances independently", () => {
      const instance1 = createMockTelem();
      const instance2 = createMockTelem();
      registerInstance("id-1", instance1);
      registerInstance("id-2", instance2);

      const result1 = factory.create({ type: "any", props: { testId: "id-1" } });
      const result2 = factory.create({ type: "any", props: { testId: "id-2" } });

      expect(result1).toBe(instance1);
      expect(result2).toBe(instance2);
      expect(result1).not.toBe(result2);
    });

    it("should have type property set to 'test'", () => {
      expect(factory.type).toBe("test");
    });
  });
});
