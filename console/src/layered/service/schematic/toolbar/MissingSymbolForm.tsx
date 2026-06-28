// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layered/service/schematic/toolbar/MissingSymbolForm.css";

import { group, type schematic } from "@synnaxlabs/client";
import {
  Button,
  Divider,
  Flex,
  Form,
  Group,
  Icon,
  Schematic,
  Text,
} from "@synnaxlabs/pluto";
import React, { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { useSymbolModal } from "@/layered/service/schematic/symbols/edit/Edit";

const SELECT_GROUP_STYLE: React.CSSProperties = { maxWidth: "60rem" };

/// MissingSymbolForm replaces the property panel for a custom-symbol node whose
/// referenced symbol cannot be resolved. The user can either pick an existing
/// symbol to re-link the node to, or create a new symbol — the group picker
/// controls which sub-group of the schematic-symbol library the new symbol is
/// parented under, matching the groups the Schematic Symbols toolbar exposes.
export const MissingSymbolForm = (): ReactElement => {
  const form = Form.useContext();
  const openEdit = useSymbolModal();
  const symbolGroup = Schematic.Symbol.useRetrieveGroup({ query: {} });
  const [createGroupKey, setCreateGroupKey] = useState<group.Key | undefined>(
    undefined,
  );
  const handleRelink = useCallback(
    (key: schematic.symbol.Key | undefined) => {
      if (key == null) return;
      form.set("specKey", key);
      form.set("stateOverrides", []);
    },
    [form],
  );
  const handleCreate = useCallback(() => {
    if (createGroupKey == null) return;
    const missingKey = form.get<string>("specKey", { optional: true })?.value;
    openEdit({
      parent: group.ontologyID(createGroupKey),
      createKey: missingKey,
    });
  }, [openEdit, createGroupKey, form]);
  return (
    <Flex.Box
      y
      align="stretch"
      gap="medium"
      className={CSS.B("schematic-missing-symbol-form")}
    >
      <Text.Text level="p" status="warning">
        <Icon.Warning />
        The custom symbol referenced by this node was not found.
      </Text.Text>
      <Schematic.Symbol.SelectSingle value={undefined} onChange={handleRelink} />
      <Divider.Divider x />
      <Flex.Box y align="stretch" gap="small">
        <Text.Text level="small" color={9}>
          Or create a new symbol in:
        </Text.Text>
        <Flex.Box x>
          {symbolGroup.data != null && (
            <Group.SelectSingle
              value={createGroupKey}
              onChange={setCreateGroupKey}
              initialQuery={{ parent: group.ontologyID(symbolGroup.data.key) }}
              style={SELECT_GROUP_STYLE}
              grow
            />
          )}
          <Button.Button
            variant="outlined"
            onClick={handleCreate}
            disabled={createGroupKey == null}
          >
            <Icon.Add />
            Create New Symbol
          </Button.Button>
        </Flex.Box>
      </Flex.Box>
    </Flex.Box>
  );
};
