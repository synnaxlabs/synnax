// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { StaticCache } from "@/telem/cache/static";

describe("StaticReadCache", () => {
  test("happy path", () => {
    const c = new StaticCache();
    const tr = TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3));
    c.write(tr, [new Series(new Float32Array([1]), DataType.FLOAT32, tr)]);
    const [res, gaps] = c.dirtyRead(
      TimeStamp.seconds(1).spanRange(TimeSpan.seconds(3))
    );
    expect(res).toHaveLength(1);
    expect(gaps).toHaveLength(0);
  });
  it("should correctly return leading and trailing gaps", () => {
    const c = new StaticCache();
    const tr = TimeStamp.seconds(2).spanRange(TimeSpan.seconds(3));
    c.write(tr, [new Series(new Float32Array([1]), DataType.FLOAT32, tr)]);
    const [res, gaps] = c.dirtyRead(
      TimeStamp.seconds(1).spanRange(TimeSpan.seconds(6))
    );
    expect(res).toHaveLength(1);
    expect(gaps).toHaveLength(2);
    expect(gaps[0].start).toEqual(TimeStamp.seconds(1));
    expect(gaps[0].end).toEqual(TimeStamp.seconds(2));
    expect(gaps[1].start).toEqual(TimeStamp.seconds(5));
    expect(gaps[1].end).toEqual(TimeStamp.seconds(7));
  });
  it("should correctly return internal gaps", () => {
    const c = new StaticCache();
    const tr1 = TimeStamp.seconds(2).spanRange(TimeSpan.seconds(3));
    const tr2 = TimeStamp.seconds(6).spanRange(TimeSpan.seconds(3));
    c.write(tr1, [new Series(new Float32Array([1]), DataType.FLOAT32, tr1)]);
    c.write(tr2, [new Series(new Float32Array([1]), DataType.FLOAT32, tr2)]);
    const [res, gaps] = c.dirtyRead(
      TimeStamp.seconds(1).spanRange(TimeSpan.seconds(7))
    );
    expect(res).toHaveLength(2);
    expect(gaps).toHaveLength(2);
    expect(gaps[0].start).toEqual(TimeStamp.seconds(1));
    expect(gaps[0].end).toEqual(TimeStamp.seconds(2));
    expect(gaps[1].start).toEqual(TimeStamp.seconds(5));
    expect(gaps[1].end).toEqual(TimeStamp.seconds(6));
  });
});
