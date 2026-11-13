// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/overview/Overview.css";

import { ranger } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Access, Panel as PlutoPanel, Ranger } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Children } from "@/feature/range/overview/Children";
import { Header } from "@/feature/range/overview/Header";
import { MetaData } from "@/feature/range/overview/MetaData";
import { AddLine } from "@/feature/range/overview/Section";
import { CSS } from "@/platform/css";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";

const NO_PAIRS: ranger.kv.Pair[] = [];

interface BodyProps {
  rangeKey: ranger.Key;
}

const Body = ({ rangeKey: key }: BodyProps): ReactElement => {
  const range = Ranger.use({ key });
  const pairs = Ranger.useResultMetaData({ rangeKey: key }).data ?? NO_PAIRS;
  const [adding, setAdding] = useState(false);
  const openCreate = Range.useCreateModal();
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const handleCreate = canCreate ? () => openCreate({ parent: key }) : undefined;
  const showMetaData = pairs.length > 0 || adding;
  return (
    <div className={CSS.BE("range-overview", "scroll")}>
      <Flex.Box y empty className={CSS.B("range-overview")}>
        <Header range={range} />
        {range.hasChildren ? (
          <Children range={range} onCreate={handleCreate} />
        ) : (
          handleCreate != null && <AddLine onClick={handleCreate}>Child range</AddLine>
        )}
        {showMetaData ? (
          <MetaData
            rangeKey={key}
            pairs={pairs}
            adding={adding}
            onAddingChange={setAdding}
          />
        ) : (
          <AddLine onClick={() => setAdding(true)}>Metadata</AddLine>
        )}
      </Flex.Box>
    </div>
  );
};

export const Overview: Panel.Content = () => {
  const { key } = PlutoPanel.useTabResource();
  // A tab can swap its range in place, so the key drops the state of the last range.
  return <Body key={key} rangeKey={key} />;
};
