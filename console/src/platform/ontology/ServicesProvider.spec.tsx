// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Ontology } from "@/platform/ontology";
import { buildServices } from "@/platform/ontology/testutil";

describe("ServicesProvider", () => {
  const channelService: Ontology.Service = {
    ...Ontology.NOOP_SERVICE,
    type: "channel",
    icon: <Icon.Channel />,
  };
  const services = buildServices({ channel: channelService });

  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Ontology.ServicesProvider services={services}>
      {children}
    </Ontology.ServicesProvider>
  );

  it("should provide the full services registry to consumers via useServices", () => {
    const { result } = renderHook(() => Ontology.useServices(), { wrapper });
    expect(result.current.channel).toBe(channelService);
  });

  it("should resolve a single service by type via useService", () => {
    const { result } = renderHook(() => Ontology.useService("channel"), { wrapper });
    expect(result.current).toBe(channelService);
    expect(result.current.hasChildren).toBe(true);
  });

  it("should throw when useServices is used outside of a provider", () => {
    expect(() => renderHook(() => Ontology.useServices())).toThrow();
  });
});
