// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/table/Table.css";

import { table } from "@synnaxlabs/client";
import { Breadcrumb } from "@synnaxlabs/lyra/breadcrumb";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import {
  Access,
  Panel as PPanel,
  type Properties,
  Staleness,
  Table,
} from "@synnaxlabs/pluto";
import { color, deep, type text } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";
import { type z } from "zod";

import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { Export } from "@/platform/export";
import { MultiEdit } from "@/platform/multiedit";
import { type Panel } from "@/platform/panel";
import { Toolbar as Base } from "@/platform/toolbar";
import { Session } from "@/session";

const Internal = (): ReactElement => {
  const key = Table.useKey();
  const name = Table.useName();
  const editable = Session.Table.useSelectEditable();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  const canEdit = hasUpdatePermission && editable;
  const selectedCellKeys = Session.Table.useSelectSelectedCellKeys();
  const cellsByKey = Table.useCells({ cellKeys: selectedCellKeys });
  const liveCellCount = cellsByKey.size;
  const singleSelectedKey =
    liveCellCount === 1 ? (cellsByKey.keys().next().value ?? null) : null;
  const selectedCellPos = Table.useCellPosition({ cellKey: singleSelectedKey ?? "" });
  const dispatch = Table.useSingleDispatch();
  const variants = new Set(Array.from(cellsByKey.values(), ({ variant }) => variant));
  const variant = variants.size === 1 ? variants.values().next().value : undefined;
  const handleVariantChange = (next: Table.Cell.Variant): void =>
    dispatch(buildVariantSwapActions(cellsByKey, next));
  return (
    <Base.Content>
      <Base.Header>
        <Flex.Box x align="center">
          <Breadcrumb.Breadcrumb>
            <Breadcrumb.Segment weight={500} color={10} level="h5">
              <Icon.Table />
              {name}
            </Breadcrumb.Segment>
            {selectedCellPos != null && (
              <Breadcrumb.Segment color={9}>
                Cell {Table.getCellColumn(selectedCellPos.x)}
                {selectedCellPos.y + 1}
              </Breadcrumb.Segment>
            )}
            {liveCellCount > 1 && (
              <Breadcrumb.Segment color={9}>{liveCellCount} cells</Breadcrumb.Segment>
            )}
          </Breadcrumb.Breadcrumb>
          {canEdit && liveCellCount > 0 && (
            <Variant value={variant} onChange={handleVariantChange} />
          )}
        </Flex.Box>
        <Flex.Box x className={CSS.BE("table", "toolbar-buttons")} gap="small">
          <Export.ToolbarButton id={table.ontologyID(key)} />
          <Core.CopyLinkToolbarButton name={name} ontologyID={table.ontologyID(key)} />
        </Flex.Box>
      </Base.Header>
      <Flex.Box full className={CSS.BE("table", "toolbar-content")}>
        {!canEdit ? (
          <NotEditableContent name={name} />
        ) : liveCellCount === 0 ? (
          <EmptyContent />
        ) : singleSelectedKey != null ? (
          // A variant's form holds that variant's config, so a swap remounts it.
          <CellForm
            key={`${singleSelectedKey}:${variant}`}
            cellKey={singleSelectedKey}
          />
        ) : (
          <MultiCellForm cellKeys={selectedCellKeys} />
        )}
      </Flex.Box>
    </Base.Content>
  );
};

interface VariantProps {
  /** The shared variant of the selected cells, or undefined when they disagree. */
  value?: Table.Cell.Variant;
  onChange: (variant: Table.Cell.Variant) => void;
}

const Variant = ({ value, onChange }: VariantProps): ReactElement => {
  const spec = value == null ? null : Table.Cell.REGISTRY[value];
  return (
    <Flex.Box x align="center" gap="small" className={CSS.BE("table", "variant")}>
      <Text.Text level="p" weight={500} color={10}>
        {spec == null ? (
          "Mixed"
        ) : (
          <>
            <spec.Icon />
            {spec.name}
          </>
        )}
      </Text.Text>
      <Table.Cell.ChangeVariant value={value} onChange={onChange} />
    </Flex.Box>
  );
};

export const Toolbar: Panel.Toolbar = () => {
  const { key } = PPanel.useTabResource();
  return (
    <Table.Suspended tableKey={key}>
      <Internal />
    </Table.Suspended>
  );
};

// buildVariantSwapActions returns one setCell action per cell whose variant
// differs from the target. Compatible fields survive the swap.
const buildVariantSwapActions = (
  cells: Iterable<[string, Table.Cell.Config]>,
  variant: Table.Cell.Variant,
): table.Action[] => {
  const spec = Table.Cell.REGISTRY[variant];
  const actions: table.Action[] = [];
  for (const [key, cell] of cells) {
    if (cell.variant === variant) continue;
    const config = deep.overrideValidItems(
      cell,
      Table.Cell.defaultConfig(variant),
      spec.schema as z.ZodType<Table.Cell.Config>,
    );
    actions.push(table.setCell({ cell: { key, config } }));
  }
  return actions;
};

interface CellFormProps {
  cellKey: string;
}

const CellForm = ({ cellKey }: CellFormProps): ReactElement | null => {
  const cell = Table.useCell({ cellKey });
  const dispatch = Table.useSingleDispatch();
  const key = Table.useKey();
  const sessionDispatch = Session.useDispatch();
  const tab = Session.Table.useSelectPropertiesTab({ key });
  const handleTabChange = (tab: Properties.TabKey) =>
    sessionDispatch(Session.Table.setPropertiesTab({ key, tab }));

  const handleChange = useCallback(
    ({ values }: Form.OnChangeParams<typeof Table.Cell.configZ>) => {
      if (cell == null) return;
      dispatch([
        table.setCell({
          cell: { key: cellKey, config: Table.Cell.configZ.parse(values) },
        }),
      ]);
    },
    [cell, cellKey, dispatch],
  );

  const methods = Form.use<typeof Table.Cell.configZ>({
    values: cell != null ? deep.copy(cell) : Table.Cell.defaultConfig("text"),
    schema: Table.Cell.configZ,
    onChange: handleChange,
    sync: true,
  });

  if (cell == null) return null;
  const C = Table.Cell.REGISTRY[cell.variant];
  return (
    <Form.Form<typeof Table.Cell.configZ> {...methods}>
      <C.Form tab={tab} onTabChange={handleTabChange} />
    </Form.Form>
  );
};

const EmptyContent = (): ReactElement => (
  <Text.Text status="disabled" center>
    No cell selected. Select a cell to view its properties.
  </Text.Text>
);

interface NotEditableContentProps {
  name: string;
}

const NotEditableContent = ({ name }: NotEditableContentProps): ReactElement => {
  const key = Table.useKey();
  const dispatch = Session.useDispatch();
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(key));
  return (
    <Empty.Action
      message={`${name} is not editable`}
      action={hasUpdatePermission ? "Enable editing" : undefined}
      onClick={() => dispatch(Session.Table.setEditable({ key, editable: true }))}
    />
  );
};

const FIELDS = MultiEdit.fieldsByVariant(table.CELL_CONFIG_SCHEMAS);

// Every stored color field that Selection colors lists.
const COLOR_FIELDS = ["textColor", "fillColor", "stalenessColor"] as const;

interface MultiCellFormProps {
  cellKeys: string[];
}

const MultiCellForm = ({ cellKeys }: MultiCellFormProps): ReactElement => {
  const cellsByKey = Table.useCells({ cellKeys });
  const dispatch = Table.useSingleDispatch();
  const theme = Theming.use();

  // One dispatch per change so undo collapses to one step.
  const selection = MultiEdit.selection({
    configs: cellsByKey,
    fields: FIELDS,
    onChange: (updates) =>
      dispatch(
        updates.map(([key, config]) =>
          table.setCell({ cell: { key, config: Table.Cell.configZ.parse(config) } }),
        ),
      ),
  });

  const selectionRefs = Array.from(cellsByKey).flatMap(([key, cell]) =>
    MultiEdit.colorRefs(key, cell, COLOR_FIELDS),
  );

  const levels = new Set(Array.from(cellsByKey.values(), (cell) => cell.level));
  const commonLevel = levels.size === 1 ? levels.values().next().value : undefined;

  return (
    <Form.Sections x>
      <Form.Section title="Colors">
        <MultiEdit.ColorField
          label="Text"
          values={selection.colors("textColor", () => theme.colors.gray.l11)}
          onChange={(c) => selection.set("textColor", c)}
        />
        <MultiEdit.ColorField
          label="Fill"
          values={selection.colors("fillColor", () => color.ZERO)}
          onChange={(c) => selection.set("fillColor", c)}
        />
        <MultiEdit.SelectionColors
          refs={selectionRefs}
          onChange={selection.setColors}
        />
      </Form.Section>
      <Form.Section title="Text">
        <Input.Item label="Size" padHelpText={false}>
          <Select.Text.Level
            value={commonLevel}
            onChange={(level: text.Level) => selection.set("level", level)}
          />
        </Input.Item>
      </Form.Section>
      {selection.has("stalenessTimeout") && (
        <MultiEdit.StalenessSection
          colors={selection.colors("stalenessColor", () =>
            Staleness.resolveColor(undefined, theme),
          )}
          timeout={selection.first("stalenessTimeout")}
          onColorChange={(c) => selection.set("stalenessColor", c)}
          onTimeoutChange={(v) => selection.set("stalenessTimeout", v)}
        />
      )}
      {selection.has("precision") && (
        <MultiEdit.NumberFormatSection
          notation={selection.first("notation")}
          precision={selection.first("precision")}
          onNotationChange={(v) => selection.set("notation", v)}
          onPrecisionChange={(v) => selection.set("precision", v)}
        />
      )}
    </Form.Sections>
  );
};
