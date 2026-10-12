// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Channel } from "@/channel";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const TRIGGER_CLASS = "pluto-channel__trigger";

describe("Channel select", () => {
  const Wrapper = createSynnaxWrapper({ client: null });

  it("should mark the single select's trigger so forms can widen it", () => {
    const c = render(<Channel.SelectSingle value={0} onChange={vi.fn()} />, {
      wrapper: Wrapper,
    });
    expect(c.container.querySelector(`.${TRIGGER_CLASS}`)).not.toBeNull();
  });

  it("should mark the multiple select's trigger so forms can widen it", () => {
    const c = render(<Channel.SelectMultiple value={[]} onChange={vi.fn()} />, {
      wrapper: Wrapper,
    });
    expect(c.container.querySelector(`.${TRIGGER_CLASS}`)).not.toBeNull();
  });
});
