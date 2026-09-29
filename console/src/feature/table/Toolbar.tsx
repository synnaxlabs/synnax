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
  Notation,
  Panel as PPanel,
  type Properties,
  Staleness,
  Table,
} from "@synnaxlabs/pluto";
import { color, deep, type notation, type record, type text } from "@synnaxlabs/x";
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

const hasField = (cell: Table.Cell.Config, field: string): boolean => {
  const fields = FIELDS.get(cell.variant);
  if (fields == null) throw new Error(`[table] - no schema for ${cell.variant}`);
  return fields.has(field);
};

// Every stored color field that Selection colors lists.
const COLOR_FIELDS = ["textColor", "fillColor", "stalenessColor"] as const;

const STALENESS_TIMEOUT_BOUNDS = { lower: 1, upper: Infinity };
const PRECISION_BOUNDS = { lower: 0, upper: 10 };

interface MultiCellFormProps {
  cellKeys: string[];
}

const MultiCellForm = ({ cellKeys }: MultiCellFormProps): ReactElement => {
  const cellsByKey = Table.useCells({ cellKeys });
  const dispatch = Table.useSingleDispatch();
  const theme = Theming.use();

  // Cells absent from the store are skipped (selection may include keys from
  // a removed row). One dispatch per call so undo collapses to one step.
  const applyConfigs = useCallback(
    (next: (key: string, cell: Table.Cell.Config) => Table.Cell.Config | null) => {
      const actions: table.Action[] = [];
      for (const key of cellKeys) {
        const cell = cellsByKey.get(key);
        if (cell == null) continue;
        const config = next(key, cell);
        if (config == null) continue;
        actions.push(
          table.setCell({ cell: { key, config: Table.Cell.configZ.parse(config) } }),
        );
      }
      dispatch(actions);
    },
    [cellKeys, cellsByKey, dispatch],
  );

  const withField = (field: string): Table.Cell.Config[] =>
    Array.from(cellsByKey.values()).filter((cell) => hasField(cell, field));

  const firstValue = <V,>(field: string): V | undefined => {
    const [cell] = withField(field);
    if (cell == null) return undefined;
    return (cell as record.Unknown)[field] as V | undefined;
  };

  // Writes the value on every selected cell that has the field.
  const handleFieldChange = (field: string, value: unknown): void =>
    applyConfigs((_, cell) =>
      hasField(cell, field) ? MultiEdit.patch(cell, [[field, value]]) : null,
    );

  const colorValues = (field: string): Array<color.Crude | undefined> =>
    withField(field).map(
      (cell) => (cell as record.Unknown)[field] as color.Crude | undefined,
    );

  const selectionRefs = Array.from(cellsByKey).flatMap(([key, cell]) =>
    MultiEdit.colorRefs(key, cell, COLOR_FIELDS),
  );

  const handleSelectionColorChange = (refs: MultiEdit.ColorRef[], c: color.Color) => {
    const byKey = MultiEdit.groupByKey(refs);
    applyConfigs((key, cell) => {
      const group = byKey.get(key);
      if (group == null) return null;
      return MultiEdit.patch(
        cell,
        group.map((r) => [r.path, c]),
      );
    });
  };

  const levels = new Set(Array.from(cellsByKey.values(), (cell) => cell.level));
  const commonLevel = levels.size === 1 ? levels.values().next().value : undefined;

  const hasStaleness = withField("stalenessColor").length > 0;
  const hasNumberFormat = withField("precision").length > 0;

  return (
    <Form.Sections x>
      <Form.Section title="Colors">
        <MultiEdit.ColorField
          label="Text"
          values={colorValues("textColor")}
          fallback={theme.colors.gray.l11}
          onChange={(c) => handleFieldChange("textColor", c)}
        />
        <MultiEdit.ColorField
          label="Fill"
          values={colorValues("fillColor")}
          fallback={color.ZERO}
          onChange={(c) => handleFieldChange("fillColor", c)}
        />
        <MultiEdit.SelectionColors
          refs={selectionRefs}
          onChange={handleSelectionColorChange}
        />
      </Form.Section>
      <Form.Section title="Text">
        <Input.Item label="Size" padHelpText={false}>
          <Select.Text.Level
            value={commonLevel}
            onChange={(level: text.Level) => handleFieldChange("level", level)}
          />
        </Input.Item>
      </Form.Section>
      {hasStaleness && (
        <Form.Section title="Staleness">
          <MultiEdit.ColorField
            label="Color"
            values={colorValues("stalenessColor")}
            fallback={Staleness.resolveColor(undefined, theme)}
            onChange={(c) => handleFieldChange("stalenessColor", c)}
          />
          <Input.Item label="Timeout" align="start" padHelpText={false}>
            <Input.Numeric
              bounds={STALENESS_TIMEOUT_BOUNDS}
              endContent="s"
              value={
                firstValue<number>("stalenessTimeout") ?? Staleness.DEFAULT_TIMEOUT
              }
              onChange={(v) => handleFieldChange("stalenessTimeout", v)}
            />
          </Input.Item>
        </Form.Section>
      )}
      {hasNumberFormat && (
        <Form.Section title="Number format">
          <Input.Item label="Notation" align="start" padHelpText={false}>
            <Notation.Select
              value={firstValue<notation.Notation>("notation") ?? "standard"}
              onChange={(v: notation.Notation) => handleFieldChange("notation", v)}
            />
          </Input.Item>
          <Input.Item label="Precision" align="start" padHelpText={false}>
            <Input.Numeric
              bounds={PRECISION_BOUNDS}
              value={firstValue<number>("precision") ?? 2}
              onChange={(v) => handleFieldChange("precision", v)}
            />
          </Input.Item>
        </Form.Section>
      )}
    </Form.Sections>
  );
};
