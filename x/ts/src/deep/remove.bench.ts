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

describe("deep.remove benchmarks", () => {
  bench("shallow property remove", () => {
    const obj = { a: 1, b: 2, c: 3, d: 4, e: 5 };
    deep.remove(obj, "c");
  });

  bench("nested property remove (5 levels)", () => {
    const obj = {
      level1: {
        level2: {
          level3: {
            level4: {
              level5: {
                value: 42,
                other: "keep",
              },
            },
          },
        },
      },
    };
    deep.remove(obj, "level1.level2.level3.level4.level5.value");
  });

  bench("array index remove", () => {
    const obj = {
      items: [1, 2, 3, 4, 5],
    };
    deep.remove(obj, "items.2");
  });

  bench("keyed array remove", () => {
    const obj = {
      items: [
        { key: "item1", value: 1 },
        { key: "item2", value: 2 },
        { key: "item3", value: 3 },
        { key: "item4", value: 4 },
        { key: "item5", value: 5 },
      ],
    };
    deep.remove(obj, "items.item3");
  });

  bench("remove from mixed array and object", () => {
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
    deep.remove(obj, "a.0.b.0.c.1");
  });

  bench("remove from large object", () => {
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

    deep.remove(obj, "key500.nested.data.name");
  });

  bench("remove from very deep path (100 levels)", () => {
    const obj: any = {};
    let current = obj;
    for (let i = 0; i < 100; i++) {
      current.next = { level: i, value: "keep" };
      current = current.next;
    }
    const path = `${new Array(100).fill("next").join(".")}.level`;
    deep.remove(obj, path);
  });

  bench("remove with keys containing periods", () => {
    const obj = {
      channels: [
        { key: "sensor.temperature.1", reading: 25.5 },
        { key: "sensor.temperature.2", reading: 26.1 },
        { key: "sensor.pressure.1", reading: 101.3 },
      ],
    };
    deep.remove(obj, "channels.sensor.temperature.1");
  });

  bench("remove non-existent key (no-op)", () => {
    const obj = {
      a: {
        b: {
          c: 1,
        },
      },
    };
    deep.remove(obj, "a.b.d");
  });

  bench("remove beyond array bounds (no-op)", () => {
    const obj = { arr: [1, 2, 3] };
    deep.remove(obj, "arr.100");
  });

  bench("remove with very deep non-existent path (no-op)", () => {
    const obj = { a: { b: 1 } };
    deep.remove(obj, "a.b.c.d.e.f.g.h.i.j.k.l.m.n.o.p");
  });

  bench("remove entire nested object", () => {
    const obj = {
      a: 1,
      b: {
        c: 2,
        d: {
          e: 3,
          f: 4,
        },
      },
      g: 5,
    };
    deep.remove(obj, "b");
  });

  bench("repeated remove operations", () => {
    const obj = { a: 1, b: 2, c: 3, d: 4, e: 5 };
    for (let i = 0; i < 5; i++) deep.remove(obj, String.fromCharCode(97 + i));
  });

  bench("remove from nested arrays", () => {
    const obj = {
      matrix: [
        [1, 2, 3],
        [4, 5, 6],
        [7, 8, 9],
      ],
    };
    deep.remove(obj, "matrix.1.1");
  });

  bench("remove with empty path (no-op)", () => {
    const obj = { a: 1, b: 2 };
    deep.remove(obj, "");
  });

  bench("remove from array with objects", () => {
    const obj = {
      users: [
        { id: 1, name: "User1", active: true },
        { id: 2, name: "User2", active: true },
        { id: 3, name: "User3", active: false },
      ],
    };
    deep.remove(obj, "users.1");
  });

  bench("remove nested property from keyed array with periods", () => {
    const obj = {
      items: [
        { key: "item.one", value: { "subitem.one": 3, "subitem.two": 4 } },
        { key: "item.two", value: 2 },
      ],
    };
    deep.remove(obj, "items.item.one.value.subitem.one");
  });

  bench("batch remove operations on different paths", () => {
    const obj = {
      user: {
        profile: {
          name: "John",
          age: 30,
          email: "john@example.com",
        },
        settings: {
          theme: "dark",
          notifications: true,
          privacy: "public",
        },
      },
    };
    deep.remove(obj, "user.profile.email");
    deep.remove(obj, "user.settings.privacy");
  });

  bench("remove from complex nested structure", () => {
    const obj = {
      app: {
        modules: [
          {
            key: "dashboard",
            components: {
              header: { props: { title: "Dashboard", logo: "logo.png" } },
              footer: { props: { copyright: "2025", links: ["home", "about"] } },
            },
          },
        ],
      },
    };
    deep.remove(obj, "app.modules.dashboard.components.footer.props.links");
  });

  bench("remove with alternating arrays and objects", () => {
    const obj = {
      data: [
        {
          items: [
            { key: "a", values: [1, 2, 3] },
            { key: "b", values: [4, 5, 6] },
          ],
        },
      ],
    };
    deep.remove(obj, "data.0.items.a.values.1");
  });
});
