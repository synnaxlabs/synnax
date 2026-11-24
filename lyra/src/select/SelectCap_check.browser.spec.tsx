// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { page } from "vitest/browser";

import { Select } from "@/select";
import { Status } from "@/status";
import { renderThemed } from "@/styletest";

// Fails each read after a load that outlasts the scroll, and starts no read while one
// runs, as a Flux list does.
const Paged = ({ reads }: { reads: () => void }): ReactElement => {
  const [status, setStatus] = useState<Status.Status>();
  const [loading, setLoading] = useState(false);
  const fetchMore = (): void => {
    if (loading) return;
    reads();
    setLoading(true);
    setStatus(Status.create({ variant: "loading", message: "Loading" }));
    setTimeout(() => {
      setLoading(false);
      setStatus(
        Status.create({
          variant: "error",
          message: "Failed to load channels",
          description: "Cannot reach the Core",
        }),
      );
    }, 500);
  };
  return (
    <Select.Single<string, undefined>
      resourceName="channel"
      value={undefined}
      onChange={() => {}}
      data={Array.from({ length: 50 }, (_, i) => `key-${i}`)}
      initialVisible
      allowNone
      status={status}
      onFetchMore={fetchMore}
    >
      {({ key, ...rest }) => (
        <Select.Item key={key} {...rest} textValue={rest.itemKey}>
          {rest.itemKey}
        </Select.Item>
      )}
    </Select.Single>
  );
};

describe("select height cap check", () => {
  afterEach(async () => {
    await page.viewport(1280, 720);
  });

  it("should read a failed page once when the window is shorter than the cap and the status", async () => {
    await page.viewport(800, 300);
    expect(window.innerHeight).toBe(300);
    const reads = vi.fn();
    const { getByRole } = await renderThemed(<Paged reads={reads} />);
    await expect.element(page.getByText("Failed to load channels")).toBeVisible();
    const scroll = getByRole("listbox").element();
    const before = reads.mock.calls.length;
    scroll.scrollTop = scroll.scrollHeight;
    await expect.poll(() => reads.mock.calls.length).toBe(before + 1);
    await sleep.sleep(TimeSpan.seconds(3));
    expect(reads).toHaveBeenCalledTimes(before + 1);
  });
});
