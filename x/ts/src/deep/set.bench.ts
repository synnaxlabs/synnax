// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bench, describe } from "vitest";

import { deep } from "@/deep";

describe("deep.set benchmarks", () => {
  bench("shallow property set", () => {
    const obj = { a: 1, b: 2, c: 3 };
    deep.set(obj, "b", 5);
  });

  bench("nested property set (5 levels)", () => {
    const obj = {
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
    deep.set(obj, "level1.level2.level3.level4.level5.value", 100);
  });

  bench("array index set", () => {
    const obj = {
      items: [1, 2, 3, 4, 5],
    };
    deep.set(obj, "items.2", 10);
  });

  bench("keyed array set", () => {
    const obj = {
      items: [
        { key: "item1", value: 1 },
        { key: "item2", value: 2 },
        { key: "item3", value: 3 },
      ],
    };
    deep.set(obj, "items.item2.value", 20);
  });

  bench("create nested path from empty object", () => {
    const obj: any = {};
    deep.set(obj, "a.b.c.d.e", "value");
  });

  bench("mixed array and object set", () => {
    const obj = {
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
    deep.set(obj, "a.0.b.0.c.1.value", 20);
  });

  bench("set in large object", () => {
    const obj: any = {};
    for (let i = 0; i < 1000; i++)
      obj[`key${i}`] = {
        nested: {
          value: i,
          data: {
            id: i,
            name: `name${i}`,
          },
        },
      };

    deep.set(obj, "key500.nested.data.name", "updated");
  });

  bench("very deep path set (100 levels)", () => {
    const obj: any = {};
    let current = obj;
    for (let i = 0; i < 99; i++) {
      current.next = { level: i };
      current = current.next;
    }
    const path = `${new Array(99).fill("next").join(".")}.value`;
    deep.set(obj, path, "deep");
  });

  bench("set with keys containing periods", () => {
    const obj = {
      channels: [
        { key: "sensor.temperature.1", reading: 25.5 },
        { key: "sensor.temperature.2", reading: 26.1 },
      ],
    };
    deep.set(obj, "channels.sensor.temperature.1.reading", 30.0);
  });

  bench("overwrite existing value", () => {
    const obj = {
      a: {
        b: {
          c: "old",
        },
      },
    };
    deep.set(obj, "a.b.c", "new");
  });

  bench("set array value beyond bounds", () => {
    const obj = { arr: [1, 2, 3] };
    deep.set(obj, "arr.10", 99);
  });

  bench("set on null values", () => {
    const obj: any = { a: null };
    deep.set(obj, "a.b.c", "value");
  });

  bench("set with negative array indices", () => {
    const obj = { arr: [1, 2, 3, 4, 5] };
    deep.set(obj, "arr.-1", 99);
  });

  bench("repeated set operations", () => {
    const obj = { a: { b: { c: 0 } } };
    for (let i = 0; i < 10; i++) deep.set(obj, "a.b.c", i);
  });

  bench("set creating arrays from numeric keys", () => {
    const obj: any = {};
    deep.set(obj, "items.0.name", "first");
    deep.set(obj, "items.1.name", "second");
    deep.set(obj, "items.2.name", "third");
  });

  bench("set on frozen object (should not throw)", () => {
    const obj = Object.freeze({ a: { b: 1 } });
    deep.set(obj, "a.c", 2);
  });

  bench("set with unicode keys", () => {
    const obj: any = {};
    deep.set(obj, "🔑.中文.العربية", "multilingual");
  });

  bench("set through proxy objects", () => {
    const target = { a: { b: 1 } };
    const proxy = new Proxy(target, {
      get(target, prop) {
        return Reflect.get(target, prop);
      },
    });
    deep.set(proxy, "a.b", 2);
  });

  bench("set on array-like objects", () => {
    const arrayLike: any = { 0: "a", 1: "b", length: 2 };
    deep.set(arrayLike, "2", "c");
    deep.set(arrayLike, "length", 3);
  });

  bench("set with mixed types in path", () => {
    const obj: any = {
      true: { false: { null: { undefined: {} } } },
    };
    deep.set(obj, "true.false.null.undefined.NaN", "special");
  });

  bench("batch set operations on different paths", () => {
    const obj = {
      user: {
        profile: {
          name: "",
          age: 0,
        },
        settings: {
          theme: "",
          notifications: false,
        },
      },
    };
    deep.set(obj, "user.profile.name", "John");
    deep.set(obj, "user.profile.age", 30);
    deep.set(obj, "user.settings.theme", "dark");
    deep.set(obj, "user.settings.notifications", true);
  });

  bench("set creating complex nested structure", () => {
    const obj: any = {};
    deep.set(obj, "app.modules.0.components.header.props.title", "My App");
    deep.set(obj, "app.modules.0.components.header.props.logo", "logo.png");
    deep.set(obj, "app.modules.0.components.footer.props.copyright", "2025");
  });
});
