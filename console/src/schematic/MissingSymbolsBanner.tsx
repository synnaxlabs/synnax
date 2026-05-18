// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/MissingSymbolsBanner.css";

import { schematic } from "@synnaxlabs/client";
import { Button, Flex, Icon, Schematic, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";

export interface MissingSymbolsBannerProps {
  layoutKey: string;
}

export const MissingSymbolsBanner = ({
  layoutKey,
}: MissingSymbolsBannerProps): ReactElement | null => {
  const refs = Schematic.useSelectCustomSymbolRefs({ key: layoutKey });
  const { missing, nodesByMissingKey } = Schematic.Symbol.useAuditCustomSymbols({
    refs,
  });
  const [expanded, setExpanded] = useState(false);
  const { update: dispatch } = Schematic.useDispatch();

  const handleRelink = useCallback(
    (oldKey: string, newKey: schematic.symbol.Key | undefined) => {
      if (newKey == null) return;
      const nodeKeys = nodesByMissingKey.get(oldKey) ?? [];
      if (nodeKeys.length === 0) return;
      dispatch({
        key: layoutKey,
        actions: nodeKeys.map((nk) =>
          schematic.setConfig({
            key: nk,
            config: { specKey: newKey, stateOverrides: [] },
          }),
        ),
      });
    },
    [dispatch, layoutKey, nodesByMissingKey],
  );

  if (missing.length === 0) return null;

  return (
    <Flex.Box
      y
      align="stretch"
      gap="tiny"
      className={CSS.B("schematic-missing-banner")}
    >
      <Flex.Box x align="center" justify="between" gap="small">
        <Flex.Box x align="center" gap="small">
          <Icon.Warning color="var(--pluto-warning-z)" />
          <Text.Text level="p">
            {missing.length === 1
              ? "1 broken symbol reference"
              : `${missing.length} broken symbol references`}
          </Text.Text>
        </Flex.Box>
        <Button.Button
          size="small"
          variant="outlined"
          onClick={() => setExpanded((p) => !p)}
        >
          {expanded ? "Hide" : "Resolve"}
        </Button.Button>
      </Flex.Box>
      {expanded && (
        <Flex.Box
          y
          align="stretch"
          gap="tiny"
          className={CSS.BE("schematic-missing-banner", "list")}
        >
          {missing.map((key) => {
            const count = nodesByMissingKey.get(key)?.length ?? 0;
            return (
              <Flex.Box
                key={key}
                x
                align="center"
                gap="small"
                className={CSS.BE("schematic-missing-banner", "row")}
              >
                <Flex.Box y align="start" gap="tiny" grow>
                  <Text.Text level="small">{key}</Text.Text>
                  <Text.Text level="small" color={8}>
                    {count === 1 ? "1 node" : `${count} nodes`}
                  </Text.Text>
                </Flex.Box>
                <Schematic.Symbol.SelectSingle
                  value={undefined}
                  onChange={(v: schematic.symbol.Key | undefined) =>
                    handleRelink(key, v)
                  }
                />
              </Flex.Box>
            );
          })}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
