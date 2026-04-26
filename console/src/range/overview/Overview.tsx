// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/overview/Overview.css";

import { Flex, Ranger } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { ChildRanges } from "@/range/overview/ChildRanges";
import { Details } from "@/range/overview/Details";
import { MetaData } from "@/range/overview/MetaData";
import { Snapshots } from "@/range/overview/Snapshots";

export const Overview: Layout.Renderer = ({ layoutKey }) => (
  <Flex.Box
    y
    style={{ padding: "5rem", maxWidth: 1050, margin: "0 auto", overflowY: "auto" }}
    className={CSS.BE("range", "overview")}
    gap="large"
  >
    <Details rangeKey={layoutKey} />
    <ChildRanges rangeKey={layoutKey} />
    <MetaData rangeKey={layoutKey} />
    <Snapshots rangeKey={layoutKey} />
  </Flex.Box>
);

const useName: Layout.NameHook = (layoutKey, onChange) => {
  const { retrieve: baseRetrieve } = Ranger.useRetrieveObservableName({
    onChange,
    addStatusOnFailure: false,
  });
  const { update } = Ranger.useRename();
  const onRename = useCallback(
    (name: string) => update({ key: layoutKey, name }),
    [layoutKey, update],
  );
  const retrieve = useCallback(
    () => baseRetrieve({ key: layoutKey }),
    [layoutKey, baseRetrieve],
  );
  return { retrieve, onRename };
};
Overview.useName = useName;
