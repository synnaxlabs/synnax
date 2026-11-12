// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/next/Range.css";

import { ranger } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Access, Panel as PlutoPanel, Ranger } from "@synnaxlabs/pluto";
import { useState } from "react";

import { Aliases } from "@/feature/range/next/overview/Aliases";
import { Children } from "@/feature/range/next/overview/Children";
import { Documents, useDocuments } from "@/feature/range/next/overview/Documents";
import { Header } from "@/feature/range/next/overview/Header";
import { MetaData } from "@/feature/range/next/overview/MetaData";
import { AddLine } from "@/feature/range/next/overview/Section";
import { CSS } from "@/platform/css";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";

type Adding = "metadata" | "alias" | null;

const NO_PAIRS: ranger.kv.Pair[] = [];

export const Overview: Panel.Content = () => {
  const { key } = PlutoPanel.useTabResource();
  const range = Ranger.use({ key });
  const pairs = Ranger.useResultMetaData({ rangeKey: key }).data ?? NO_PAIRS;
  const aliases = Ranger.useResultAliases({ rangeKey: key }).data;
  const documents = useDocuments(key);
  const [adding, setAdding] = useState<Adding>(null);
  const openCreate = Range.useCreateModal();
  const canCreate = Access.useCreateGranted(ranger.TYPE_RESOURCE_ID);
  const handleCreate = canCreate ? () => openCreate({ parent: key }) : undefined;
  const showMetaData = pairs.length > 0 || adding === "metadata";
  const aliasCount = aliases == null ? 0 : Object.keys(aliases).length;
  const showAliases = aliasCount > 0 || adding === "alias";
  const setAddingTo = (to: Adding) => (next: boolean) => setAdding(next ? to : null);
  return (
    <div key={key} className={CSS.BE("range-overview", "scroll")}>
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
            adding={adding === "metadata"}
            onAddingChange={setAddingTo("metadata")}
          />
        ) : (
          <AddLine onClick={() => setAdding("metadata")}>Metadata</AddLine>
        )}
        {aliases != null &&
          (showAliases ? (
            <Aliases
              rangeKey={key}
              parent={range.parent}
              aliases={aliases}
              adding={adding === "alias"}
              onAddingChange={setAddingTo("alias")}
            />
          ) : (
            <AddLine onClick={() => setAdding("alias")}>Channel alias</AddLine>
          ))}
        {documents.documents.length > 0 && <Documents {...documents} />}
      </Flex.Box>
    </div>
  );
};
