// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient, TimeRange, TimeSpan } from "@synnaxlabs/client";
import { renderHook, waitFor } from "@testing-library/react";
import { act, type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Sync } from "@/query/sync";
import { Ranger } from "@/ranger";
import { Synnax } from "@/synnax";

const client = newTestClient();
const Wrapper = (props: PropsWithChildren) => (
  <Synnax.TestProvider client={client}>
    <Sync.Provider {...props} />
  </Synnax.TestProvider>
);

describe("queries", () => {
  describe("use", () => {
    it("should retrieve a range", async () => {
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(10)),
      });
      const { result } = renderHook(() => Ranger.use(rng.key), { wrapper: Wrapper });
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
        expect(result.current.data?.key).toEqual(rng.key);
      });
    });

    it("should update the range", async () => {
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(10)),
      });
      const { result } = renderHook(() => Ranger.use(rng.key), {
        wrapper: Wrapper,
      });
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
        expect(result.current.data?.key).toEqual(rng.key);
        expect(result.current.data?.name).toEqual("test");
      });
      await client.ranges.create({
        key: rng.key,
        name: "updated test",
        timeRange: new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(10)),
      });
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
        expect(result.current.data?.key).toEqual(rng.key);
        expect(result.current.data?.name).toEqual("updated test");
      });
    });
  });

  describe("useForm", () => {
    it("should allow the caller to update the range", async () => {
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(10)),
      });
      const formValues = await Ranger.rangeToFormValues(rng);
      const { result } = renderHook(
        () =>
          Ranger.useForm({
            autoSave: false,
            params: rng.key,
            initialValues: formValues,
          }),
        { wrapper: Wrapper },
      );
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
      });
      await act(async () => {
        result.current.form.set("name", "new name");
        result.current.save();
      });
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
        expect(result.current.form.get("name").value).toEqual("new name");
        const rng2 = await client.ranges.retrieve(rng.key);
        expect(rng2.name).toEqual("new name");
      });
    });

    it.only("should allow the caller to add labels to the range", async () => {
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(10)),
      });
      const label = await client.labels.create({
        name: "test",
        color: "#000000",
      });
      const formValues = await Ranger.rangeToFormValues(rng);
      const { result } = renderHook(
        () =>
          Ranger.useForm({
            autoSave: false,
            params: rng.key,
            initialValues: formValues,
          }),
        { wrapper: Wrapper },
      );
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
      });
      await act(async () => {
        result.current.form.set("labels", [label.key]);
        result.current.save();
      });
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
        expect(result.current.form.get("labels").value).toEqual([label.key]);
        expect(await rng.labels()).toEqual([label]);
      });
    });

    it("should allow the caller to set the parent range", async () => {
      const rng = await client.ranges.create({
        name: "test",
        timeRange: new TimeRange(TimeSpan.seconds(5), TimeSpan.seconds(10)),
      });
      const parent = await client.ranges.create({
        name: "parent",
        timeRange: new TimeRange(TimeSpan.seconds(3), TimeSpan.seconds(7)),
      });
      const formValues = await Ranger.rangeToFormValues(rng);
      const { result } = renderHook(
        () =>
          Ranger.useForm({
            autoSave: false,
            params: rng.key,
            initialValues: formValues,
          }),
        { wrapper: Wrapper },
      );
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
      });
      await act(async () => {
        result.current.form.set("parent", parent.key);
        result.current.save();
      });
      await waitFor(async () => {
        expect(result.current.status).toEqual("success");
        expect(result.current.form.get("parent").value).toEqual(parent.key);
        expect(await rng.retrieveParent()).toEqual(parent);
      });
      await new Promise((resolve) => setTimeout(resolve, 1000));
    });
  });
});
