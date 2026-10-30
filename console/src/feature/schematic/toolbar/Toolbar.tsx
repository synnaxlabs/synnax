// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/toolbar/Toolbar.css";

import { schematic } from "@synnaxlabs/client";
import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Tabs } from "@synnaxlabs/lyra/tabs";
import { Text } from "@synnaxlabs/lyra/text";
import { Panel as PlutoPanel, Schematic } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Control } from "@/feature/schematic/toolbar/Control";
import { Properties } from "@/feature/schematic/toolbar/Properties";
import { Symbols, type SymbolsChangeProps } from "@/feature/schematic/toolbar/Symbols";
import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Errors } from "@/platform/errors";
import { Export } from "@/platform/export";
import { Toolbar as Base } from "@/platform/toolbar";
import { Session } from "@/session";

const NotEditableContent = (): ReactElement => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const controlState = Session.Schematic.useSelectControlStatus();
  const { canEdit } = Session.Schematic.useSelectEditable();
  const name = Schematic.useName();
  return (
    <Empty.Action
      message={`${name} is not editable`}
      action={
        canEdit
          ? controlState === "acquired"
            ? "Release control and enable editing"
            : "Enable editing"
          : undefined
      }
      onClick={() => {
        dispatch(Session.Schematic.setEditable({ key, editable: true }));
      }}
    />
  );
};

// Group boxes come only from grouping, and a locked group shields its members.
const useChangeableSymbols = (
  selected: string[],
): Map<string, Schematic.Node.Config> => {
  const configs = Schematic.useConfigs({ keys: selected });
  const parentOf = Schematic.useParentOf();
  const shielded = Schematic.Group.shielded(
    [...configs.keys()],
    parentOf,
    Object.fromEntries(configs),
  );
  const out = new Map<string, Schematic.Node.Config>();
  configs.forEach((config, key) => {
    if (
      Schematic.Node.isConfig(config) &&
      config.variant !== Schematic.Node.GroupBox.VARIANT &&
      !shielded.has(key)
    )
      out.set(key, config);
  });
  return out;
};

interface SymbolName {
  /** The symbol that every config draws, if they all draw the same one. */
  symbol?: string;
  name: string;
}

const useSymbolName = (configs: Map<string, Schematic.Node.Config>): SymbolName => {
  const symbolKeys = new Set(Array.from(configs.values(), Schematic.Node.symbolKey));
  const shared = symbolKeys.size === 1 ? configs.values().next().value : undefined;
  const specKey =
    shared != null && Schematic.Node.isCustomConfig(shared) ? shared.specKey : null;
  const { symbol: custom } = Schematic.Symbol.useResolved(specKey);
  if (shared == null) return { name: "Mixed" };
  if (specKey != null) return { symbol: specKey, name: custom?.name ?? "" };
  const spec = Schematic.Node.resolveSpec(shared.variant);
  return { symbol: shared.variant, name: spec.label ?? spec.name };
};

interface SymbolSummaryProps {
  name: string;
  changing: boolean;
  onChangingChange: (changing: boolean) => void;
}

const SymbolSummary = ({
  name,
  changing,
  onChangingChange,
}: SymbolSummaryProps): ReactElement => (
  <Flex.Box x align="center" gap="small" className={CSS.BE("schematic", "symbol")}>
    {changing ? (
      <>
        <Flex.Box
          x
          align="center"
          gap="small"
          className={CSS.BE("schematic", "change")}
        >
          <Text.Text level="small" weight={500} color={11}>
            Changing {name}
          </Text.Text>
          <Text.Text level="small" color={9}>
            Click a symbol to change to it
          </Text.Text>
        </Flex.Box>
        <Button.Button
          variant="outlined"
          size="small"
          aria-label="Cancel symbol change"
          onClick={() => onChangingChange(false)}
        >
          Cancel
        </Button.Button>
      </>
    ) : (
      <>
        <Text.Text level="p" weight={500} color={10}>
          {name}
        </Text.Text>
        <Button.Button
          variant="text"
          size="small"
          textColor="var(--pluto-primary-p1)"
          aria-label="Change symbol"
          onClick={() => onChangingChange(true)}
        >
          Change
        </Button.Button>
      </>
    )}
  </Flex.Box>
);

const Internal = (): ReactElement => {
  const key = Schematic.useKey();
  const dispatch = Session.useDispatch();
  const activeTab = Session.Schematic.useSelectActiveToolbarTab();
  const name = Schematic.useName();
  const { isCurrentlyEditable, canEdit } = Session.Schematic.useSelectEditable();
  const selected = Session.Schematic.useSelectSelected();
  const changeable = useChangeableSymbols(selected);
  const current = useSymbolName(changeable);
  const changing = Session.Schematic.useSelectChanging();
  const changeSymbol = Schematic.useChangeSymbol();
  const setChanging = (changing: boolean) =>
    dispatch(Session.Schematic.setChanging({ key, changing }));
  const singleSelectedConfig = Schematic.useElementConfig({
    elKey: selected.length === 1 ? selected[0] : "",
  });
  const change: SymbolsChangeProps | undefined = changing
    ? {
        value: current.symbol,
        onChange: (input) => {
          changeSymbol([...changeable.keys()], input);
          setChanging(false);
        },
      }
    : undefined;
  const singleSelectedName =
    selected.length === 1 &&
    singleSelectedConfig != null &&
    "label" in singleSelectedConfig
      ? (singleSelectedConfig.label?.label ?? null)
      : null;
  const handleTabSelect = (tabKey: string): void => {
    dispatch(
      Session.Schematic.selectToolbarTab({
        key,
        tab: tabKey as Session.Schematic.ToolbarTab,
      }),
    );
  };
  return (
    <Base.Content>
      <Tabs.Frame value={activeTab} onChange={handleTabSelect} grow>
        <Base.Header>
          <Flex.Box x align="center">
            <Breadcrumb.Breadcrumb level="h5">
              <Base.TitleSegment>
                <Icon.Schematic />
                {name}
              </Base.TitleSegment>
              {singleSelectedName !== null && (
                <Breadcrumb.Segment weight={400} color={9} level="small">
                  {singleSelectedName}
                </Breadcrumb.Segment>
              )}
            </Breadcrumb.Breadcrumb>
            {isCurrentlyEditable && changeable.size > 0 && (
              <SymbolSummary
                name={current.name}
                changing={changing}
                onChangingChange={setChanging}
              />
            )}
          </Flex.Box>
          <Flex.Box x align="center" empty>
            <Flex.Box
              x
              gap="small"
              className={CSS.BE("schematic", "toolbar", "actions")}
            >
              <Export.ToolbarButton id={schematic.ontologyID(key)} />
              <Core.CopyLinkToolbarButton
                name={name}
                ontologyID={schematic.ontologyID(key)}
              />
            </Flex.Box>
            {canEdit && (
              <Tabs.Selector className={CSS.BE("schematic", "toolbar", "tab-selector")}>
                <Tabs.Tab itemKey="symbols">Symbols</Tabs.Tab>
                <Tabs.Tab itemKey="properties">Properties</Tabs.Tab>
                <Tabs.Tab itemKey="control">Control</Tabs.Tab>
              </Tabs.Selector>
            )}
          </Flex.Box>
        </Base.Header>
        {isCurrentlyEditable ? (
          <>
            <Tabs.Content itemKey="symbols">
              <Errors.SuspenseBoundary>
                <Symbols change={change} />
              </Errors.SuspenseBoundary>
            </Tabs.Content>
            <Tabs.Content itemKey="properties">
              <Errors.SuspenseBoundary>
                <Properties />
              </Errors.SuspenseBoundary>
            </Tabs.Content>
            <Tabs.Content itemKey="control">
              <Errors.SuspenseBoundary>
                <Control />
              </Errors.SuspenseBoundary>
            </Tabs.Content>
          </>
        ) : (
          <Tabs.Content>
            <NotEditableContent />
          </Tabs.Content>
        )}
      </Tabs.Frame>
    </Base.Content>
  );
};

export const Toolbar = (): ReactElement => {
  const { key } = PlutoPanel.useTabResource();
  return (
    <Schematic.Suspended schematicKey={key}>
      <Internal />
    </Schematic.Suspended>
  );
};
