// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Analytics } from "@/platform/analytics";
import { createTestSink } from "@/platform/analytics/testutil";
import { Arc } from "@/platform/arc";

const client = createTestClient();

const renderCaptureDeploy = () => {
  const analytics = createTestSink();
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Analytics.Provider sink={analytics}>{children}</Analytics.Provider>
  );
  const { result } = renderHook(Arc.useCaptureDeploy, { wrapper });
  return { analytics, captureDeploy: result.current };
};

describe("useCaptureDeploy", () => {
  it("should report each automation a command batch started", () => {
    const { analytics, captureDeploy } = renderCaptureDeploy();
    captureDeploy({
      client,
      data: [
        { task: "1", type: "start" },
        { task: "2", type: "start" },
      ],
    });
    expect(analytics.capture.mock.calls).toEqual([
      ["automation_deployed", {}],
      ["automation_deployed", {}],
    ]);
  });

  it("should not report an automation that was stopped", () => {
    const { analytics, captureDeploy } = renderCaptureDeploy();
    captureDeploy({ client, data: { task: "1", type: "stop" } });
    expect(analytics.capture).not.toHaveBeenCalled();
  });
});
