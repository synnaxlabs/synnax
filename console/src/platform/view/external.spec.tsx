// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, UnexpectedError, view } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Flux } from "@synnaxlabs/pluto";
import { uniqueName } from "@synnaxlabs/x/testutil";
import {
  act,
  renderHook,
  type RenderHookResult,
  waitFor,
} from "@testing-library/react";
import { Component, type PropsWithChildren, type ReactNode } from "react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { View } from "@/platform/view";
import { createTabWrapper } from "@/platform/view/testutil";
import { renderSuspended } from "@/testutil";

const client = createTestClient();

const EXPLORER: panel.NewTab = { variant: "view", type: "explorer" };

/**
 * Renders hook in a range frame in tab, resolving once the render commits. A frame of a
 * stored view suspends while it reads the view.
 */
const renderInFrame = async <R,>(
  hook: () => R,
  tab: panel.NewTab = EXPLORER,
): Promise<RenderHookResult<R, unknown>> => {
  const { wrapper: Tab } = await createTabWrapper({ client, tab });
  const Wrapper = ({ children }: PropsWithChildren) => (
    <Tab>
      <View.Frame resourceType="range" icon="Range" tabType="explorer">
        {children}
      </View.Frame>
    </Tab>
  );
  let rendered!: RenderHookResult<R, unknown>;
  await act(async () => {
    rendered = renderHook(hook, { wrapper: Wrapper });
  });
  return rendered;
};

/** Creates a range view that is deleted when the test ends. */
const createView = async (layout: view.RangeView["layout"]): Promise<view.View> => {
  const created = await client.views.create({
    name: uniqueName("view"),
    type: "range",
    sort: "newest",
    layout,
  });
  onTestFinished(async () => await client.views.delete(created.key));
  return created;
};

const showing = (saved: view.View): panel.NewTab => ({
  variant: "resource",
  resource: view.resourceID(saved.key),
});

/** Holds the error that its children throw while they render. */
class Catch extends Component<
  PropsWithChildren<{ onError: (error: Error) => void }>,
  { failed: boolean }
> {
  override state = { failed: false };

  static getDerivedStateFromError(): { failed: boolean } {
    return { failed: true };
  }

  override componentDidCatch(error: Error): void {
    this.props.onError(error);
  }

  override render(): ReactNode {
    return this.state.failed ? null : this.props.children;
  }
}

describe("View", () => {
  describe("useInitialView", () => {
    it("should return the view that the frame opens with", async () => {
      const { result } = await renderInFrame(() => View.useInitialView("range"));
      await waitFor(() =>
        expect(result.current).toMatchObject({
          type: "range",
          sort: "newest",
          layout: "tree",
        }),
      );
    });

    it("should throw for a view of another type", async () => {
      const errors: unknown[] = [];
      const useStatus = () => {
        try {
          return View.useInitialView("status");
        } catch (e) {
          errors.push(e);
          return null;
        }
      };
      await renderInFrame(useStatus);
      await waitFor(() => expect(errors).not.toHaveLength(0));
      expect(errors[0]).toBeInstanceOf(UnexpectedError);
      expect((errors[0] as Error).message).toContain(
        "Expected a status view, got a range view",
      );
    });
  });

  describe("useField", () => {
    it("should hold a change to a field of a static view", async () => {
      const { result } = await renderInFrame(() => View.useField("range", "layout"));
      await waitFor(() => expect(result.current?.[0]).toBe("tree"));
      act(() => result.current[1]("list"));
      await waitFor(() => expect(result.current[0]).toBe("list"));
    });

    it("should read a field of a stored view and save a change to it", async () => {
      const saved = await createView("list");
      const { result } = await renderInFrame(
        () => View.useField("range", "layout"),
        showing(saved),
      );
      await waitFor(() => expect(result.current?.[0]).toBe("list"));
      await act(async () => result.current[1]("tree"));
      await waitFor(async () => {
        const stored = await client.views.retrieve({ key: saved.key });
        expect(stored).toMatchObject({ type: "range", layout: "tree" });
      });
      await waitFor(() => expect(result.current[0]).toBe("tree"));
    });
  });

  describe("Form", () => {
    it("should throw when the frame holds views of another resource type", async () => {
      // React reports the error that the boundary catches.
      vi.spyOn(console, "error").mockImplementation(() => {});
      const { wrapper: Tab } = await createTabWrapper({ client, tab: EXPLORER });
      const StatusForm = () => {
        const listProps = Flux.useList((client) => client.statuses, {
          name: "statuses",
          query: {},
        });
        return (
          <View.Form
            type="status"
            {...listProps}
            onWhereChange={() => {}}
            itemHeight={40}
          />
        );
      };
      const errors: Error[] = [];
      await renderSuspended(
        <Tab>
          <View.Frame resourceType="range" icon="Range" tabType="explorer">
            <Catch onError={(e) => errors.push(e)}>
              <StatusForm />
            </Catch>
          </View.Frame>
        </Tab>,
      );
      await waitFor(() => expect(errors).not.toHaveLength(0));
      expect(errors[0]).toBeInstanceOf(UnexpectedError);
      expect(errors[0].message).toContain(
        "A status form cannot show the views of a range frame",
      );
    });
  });
});
