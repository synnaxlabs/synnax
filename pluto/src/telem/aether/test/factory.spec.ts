// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { type Telem } from "@/telem/aether/telem";
import { registerInstance, TestFactory } from "@/telem/aether/test/factory";

const createMockTelem = (): Telem => ({
  cleanup: () => {},
});

describe("TestFactory", () => {
  describe("registerTestInstance and unregisterTestInstance", () => {
    it("should register an instance that can be retrieved by factory", () => {
      const instance = createMockTelem();
      const id = "test-id-1";

      const unregister = registerInstance(id, instance);

      const factory = new TestFactory();
      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "source",
        props: { testId: id },
      });

      expect(result).toBe(instance);

      unregister();
    });

    it("should unregister an instance so factory returns null", () => {
      const instance = createMockTelem();
      const id = "test-id-2";

      const unregister = registerInstance(id, instance);
      unregister();

      const factory = new TestFactory();
      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "source",
        props: { testId: id },
      });

      expect(result).toBeNull();
    });
  });

  describe("create", () => {
    let factory: TestFactory;
    const destructors: destructor.Destructor[] = [];

    beforeEach(() => {
      factory = new TestFactory();
    });

    afterEach(() => {
      destructors.forEach((destructor) => destructor());
    });

    const registerAndTrack = (id: string, instance: Telem): destructor.Destructor => {
      const unregister = registerInstance(id, instance);
      destructors.push(unregister);
      return unregister;
    };

    it("should return null for spec without testId", () => {
      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "sink",
        props: {},
      });
      expect(result).toBeNull();
    });

    it("should return null for spec without props", () => {
      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "sink",
        props: {},
      });
      expect(result).toBeNull();
    });

    it("should return null for unregistered testId", () => {
      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "sink",
        props: { testId: "non-existent" },
      });
      expect(result).toBeNull();
    });

    it("should return registered instance for valid testId", () => {
      const instance = createMockTelem();
      const unregister = registerAndTrack("valid-id", instance);

      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "sink",
        props: { testId: "valid-id" },
      });

      expect(result).toBe(instance);
      unregister();
    });

    it("should handle multiple instances independently", () => {
      const instance1 = createMockTelem();
      const instance2 = createMockTelem();
      const unregister1 = registerAndTrack("id-1", instance1);
      const unregister2 = registerAndTrack("id-2", instance2);

      const result1 = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "source",
        props: { testId: "id-1" },
      });
      const result2 = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "source",
        props: { testId: "id-2" },
      });

      expect(result1).toBe(instance1);
      expect(result2).toBe(instance2);
      expect(result1).not.toBe(result2);
      unregister1();
      unregister2();
    });

    it("should have type property set to 'test'", () => {
      expect(factory.type).toBe("test");
    });
  });
});
