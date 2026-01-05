// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bench, describe } from "vitest";

import { deep } from "@/deep";

describe("deep.get benchmarks", () => {
  const shallowObject = {
    a: 1,
    b: 2,
    c: 3,
    d: 4,
    e: 5,
  };

  const nestedObject = {
    level1: {
      level2: {
        level3: {
          level4: {
            level5: {
              value: 42,
            },
          },
        },
      },
    },
  };

  const arrayObject = {
    items: [
      { key: "item1", value: 1 },
      { key: "item2", value: 2 },
      { key: "item3", value: 3 },
      { key: "item4", value: 4 },
      { key: "item5", value: 5 },
    ],
  };

  const deeplyNestedArray = {
    a: [
      {
        b: [
          {
            c: [{ value: 1 }, { value: 2 }, { value: 3 }],
          },
        ],
      },
    ],
  };

  const largeObject: any = {};
  for (let i = 0; i < 1000; i++)
    largeObject[`key${i}`] = {
      nested: {
        value: i,
        data: {
          id: i,
          name: `name${i}`,
        },
      },
    };

  const veryDeepObject: any = {};
  let current = veryDeepObject;
  for (let i = 0; i < 100; i++) {
    current.next = { level: i };
    current = current.next;
  }
  current.value = "deep";

  const keyedWithPeriods = {
    channels: [
      { key: "sensor.temperature.1", reading: 25.5 },
      { key: "sensor.temperature.2", reading: 26.1 },
      { key: "sensor.pressure.1", reading: 101.3 },
      { key: "sensor.pressure.2", reading: 101.5 },
    ],
  };

  bench("shallow property access", () => {
    deep.get(shallowObject, "c");
  });

  bench("nested property access (5 levels)", () => {
    deep.get(nestedObject, "level1.level2.level3.level4.level5.value");
  });

  bench("array index access", () => {
    deep.get(arrayObject, "items.2");
  });

  bench("keyed array access", () => {
    deep.get(arrayObject, "items.item3.value");
  });

  bench("mixed array and object access", () => {
    deep.get(deeplyNestedArray, "a.0.b.0.c.1.value");
  });

  bench("large object property access", () => {
    deep.get(largeObject, "key500.nested.data.name");
  });

  bench("very deep path (100 levels)", () => {
    const path = `${new Array(99).fill("next").join(".")}.value`;
    deep.get(veryDeepObject, path);
  });

  bench("optional path that doesn't exist", () => {
    deep.get(nestedObject, "level1.nonexistent.path", { optional: true });
  });

  bench("keyed array with periods in keys", () => {
    deep.get(keyedWithPeriods, "channels.sensor.temperature.1.reading");
  });

  bench("has() on existing path", () => {
    deep.has(nestedObject, "level1.level2.level3");
  });

  bench("has() on non-existing path", () => {
    deep.has(nestedObject, "level1.level2.nonexistent");
  });

  bench("has() on keyed array", () => {
    deep.has(arrayObject, "items.item2");
  });

  const pathsWith100Keys: any = {};
  let temp = pathsWith100Keys;
  for (let i = 0; i < 100; i++) {
    temp[`key_${i}`] = {};
    temp = temp[`key_${i}`];
  }
  temp.final = "value";

  bench("100 unique keys traversal", () => {
    const path = Array.from({ length: 100 }, (_, i) => `key_${i}`).join(".");
    deep.get(pathsWith100Keys, `${path}.final`);
  });

  bench("repeated get operations (cache test)", () => {
    for (let i = 0; i < 10; i++)
      deep.get(nestedObject, "level1.level2.level3.level4.level5.value");
  });

  bench("get with custom getter function", () => {
    const obj = {
      data: {
        value: () => ({ result: 42 }),
      },
    };
    deep.get(obj, "data.value().result", {
      optional: false,
      getter: (obj, key) => {
        if (key === "value()")
          return (obj as { value: () => { result: number } }).value();
        return obj[key];
      },
    });
  });
});
