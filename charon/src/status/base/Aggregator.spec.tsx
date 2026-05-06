// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { act, render, renderHook } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Status } from "@/status/base";

const wrapper = ({ children }: PropsWithChildren) => (
  <Status.Aggregator>{children}</Status.Aggregator>
);

describe("Aggregator", () => {
  it("should correctly render an aggregator", () => {
    const { getByText } = render(
      <Status.Aggregator>
        <p>Test</p>
      </Status.Aggregator>,
    );
    expect(getByText("Test")).toBeTruthy();
  });

  describe("add", () => {
    it("should allow the caller to add a status", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].message).toEqual("Test");
    });
    it("should allow the caller to silence a notification", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const key = result.current.statuses.statuses[0].key;
      act(() => {
        result.current.statuses.silence(key);
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });
    it("should remove notifications after an expiration threshold", async () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications({
            expiration: TimeSpan.milliseconds(1),
            poll: TimeSpan.milliseconds(1),
          }),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      await expect
        .poll(async () => result.current.statuses.statuses.length === 0)
        .toBeTruthy();
    });
  });

  describe("silence", () => {
    it("should silence a notification", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      const key = result.current.statuses.statuses[0].key;
      act(() => {
        result.current.statuses.silence(key);
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });

    it("should silence all notifications with the same message and variant", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useAdder(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      // Add multiple notifications with same message and variant
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test 1",
        });
      });
      act(() => {
        result.current.add({
          variant: "success",
          message: "Test",
          description: "Test 2",
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].count).toBe(2);

      const key = result.current.statuses.statuses[0].key;
      act(() => {
        result.current.statuses.silence(key);
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });
  });

  describe("handleError", () => {
    it("should create a status from an exception", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add(new Error("Test"));
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].message).toEqual("Test");
    });
    it("should allow the caller to provide a custom message", () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      act(() => {
        result.current.add(new Error("Test"), "Custom");
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].message).toEqual("Custom");
    });
    it("should allow the caller to pass in an async function", async () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      await act(async () => {
        await new Promise<void>((resolve) => {
          result.current.add(async () => {
            resolve();
            throw new Error("Test");
          });
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(1);
      expect(result.current.statuses.statuses[0].message).toEqual("Test");
    });
    it("should not add a status if no error is thrown", async () => {
      const { result } = renderHook(
        () => ({
          add: Status.useErrorHandler(),
          statuses: Status.useNotifications(),
        }),
        { wrapper },
      );
      await act(async () => {
        await new Promise<void>((resolve) => {
          result.current.add(async () => {
            resolve();
          });
        });
      });
      expect(result.current.statuses.statuses).toHaveLength(0);
    });
  });
});
