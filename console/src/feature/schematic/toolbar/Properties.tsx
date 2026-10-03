// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import {
  Diagram,
  Direction,
  type Properties as PProperties,
  Schematic,
  Staleness,
} from "@synnaxlabs/pluto";
import {
  box,
  color,
  deep,
  type dimensions,
  type direction,
  location,
  type text,
  xy,
} from "@synnaxlabs/x";
import {
  type FC,
  memo,
  type ReactElement,
  type ReactNode,
  useCallback,
  useMemo,
} from "react";
import { z } from "zod";

import { Symbol } from "@/feature/schematic/symbol";
import { CSS } from "@/platform/css";
import { MultiEdit } from "@/platform/multiedit";
import { Session } from "@/session";

export const Properties = memo((): ReactElement => {
  const selected = Session.Schematic.useSelectSelected();
  const configByKey = Schematic.useConfigs({ keys: selected });
  if (selected.length === 0 || configByKey.size === 0)
    return (
      <Text.Text status="disabled" center>
        Select a schematic element to configure its properties
      </Text.Text>
    );
  if (selected.length > 1) return <MultiConfig configByKey={configByKey} />;
  const elKey = selected[0];
  return <IndividualConfig key={elKey} elKey={elKey} />;
});
Properties.displayName = "PropertiesControls";

interface IndividualConfigProps {
  elKey: string;
}

const IndividualConfig = ({ elKey }: IndividualConfigProps): ReactElement | null => {
  const config = Schematic.useElementConfig({ elKey });
  if (config == null) throw new Error(`Element with key ${elKey} not found`);
  const schematicKey = Schematic.useKey();
  const dispatch = Schematic.useSingleDispatch();
  const sessionDispatch = Session.useDispatch();
  const tab = Session.Schematic.useSelectPropertiesTab({ key: schematicKey });
  const handleTabChange = (tab: PProperties.TabKey) =>
    sessionDispatch(Session.Schematic.setPropertiesTab({ key: schematicKey, tab }));
  const initialValues = useMemo(() => deep.copy(config), [config]);
  const formMethods = Form.use<typeof Schematic.elementConfigZ>({
    schema: Schematic.elementConfigZ,
    values: initialValues,
    sync: true,
    onChange: useCallback(
      ({ values }: Form.OnChangeParams<typeof Schematic.elementConfigZ>) =>
        dispatch(schematic.setConfig({ key: elKey, config: deep.copy(values) })),
      [dispatch, elKey],
    ),
  });
  const specKey = Form.useFieldValue<string, string, typeof Schematic.elementConfigZ>(
    "specKey",
    { ctx: formMethods, optional: true },
  );
  const isCustom = z.validate(schematic.symbol.keyZ, specKey) && specKey != null;
  const openEditModal = Symbol.Edit.useModal();
  let actions: ReactNode = null;
  if (isCustom)
    actions = (
      <Button.Button
        variant="filled"
        size="tiny"
        className={CSS.BE("schematic", "properties", "edit")}
        onClick={() => openEditModal({ symbolKey: specKey })}
      >
        <Icon.Edit />
      </Button.Button>
    );

  if (config == null) return null;
  const C = Schematic.ELEMENT_REGISTRY[config.variant];
  const formProps: Schematic.Node.FormProps = {
    actions,
    schematicKey,
    tab,
    onTabChange: handleTabChange,
  };
  return (
    <Flex.Box className={CSS.BE("schematic", "properties")} y>
      <Form.Form<typeof Schematic.elementConfigZ> {...formMethods}>
        {isCustom ? (
          <CustomVariantForm
            specKey={specKey}
            elKey={elKey}
            formProps={formProps}
            VariantForm={C.Form}
          />
        ) : (
          <C.Form key={elKey} {...formProps} />
        )}
      </Form.Form>
    </Flex.Box>
  );
};

interface CustomVariantFormProps {
  specKey: string;
  elKey: string;
  formProps: Schematic.Node.FormProps;
  VariantForm: FC<Schematic.Node.FormProps>;
}

const CustomVariantForm = ({
  specKey,
  elKey,
  formProps,
  VariantForm,
}: CustomVariantFormProps): ReactElement => {
  const { missing } = Schematic.Symbol.useResolved(specKey);
  if (missing) return <Symbol.MissingForm />;
  return <VariantForm key={elKey} {...formProps} />;
};

const FIELDS = MultiEdit.fieldsByVariant(schematic.ELEMENT_CONFIG_SCHEMAS);

// Every stored color field, role or not, that Selection lists.
const COLOR_FIELDS = [
  "strokeColor",
  "fillColor",
  "textColor",
  "levelColor",
  "axisColor",
  "onColor",
  "stalenessColor",
] as const;

const selectionColorRefs = (
  key: string,
  cfg: Schematic.ElementConfig,
): MultiEdit.ColorRef[] => {
  const refs = MultiEdit.colorRefs(key, cfg, COLOR_FIELDS);
  if (cfg.variant === "state_indicator")
    cfg.options.forEach((o, i) => {
      if (o.color != null)
        refs.push({ key, path: `options.${i}.color`, value: color.construct(o.color) });
    });
  return refs;
};

const SCALE_BOUNDS = { lower: 5, upper: 1000 };

interface MultiElementPropertiesProps {
  configByKey: Map<string, Schematic.ElementConfig>;
}

const MultiConfig = ({ configByKey }: MultiElementPropertiesProps): ReactElement => {
  const schematicKey = Schematic.useKey();
  const handleError = Status.useErrorHandler();
  const selected = Session.Schematic.useSelectSelected();
  const selectedNodes = Schematic.useNodes({ keys: selected });
  const dispatch = Schematic.useSingleDispatch();
  const getViewport = Session.Schematic.useGetViewport();
  const parentOf = Schematic.useParentOf();
  // Grouped symbols never move alone: boxes align, the dispatch fans out to members.
  const movable = useMemo(
    () => selected.filter((k) => !parentOf.has(k)),
    [selected, parentOf],
  );
  // The selection closure includes group configs, so shielding sees locked flags.
  const editableByKey = useMemo(() => {
    const shielded = Schematic.Group.shielded(
      [...configByKey.keys()],
      parentOf,
      Object.fromEntries(configByKey),
    );
    if (shielded.size === 0) return configByKey;
    const m = new Map(configByKey);
    shielded.forEach((k) => m.delete(k));
    return m;
  }, [configByKey, parentOf]);

  const nodesByKey = useMemo(() => {
    const m = new Map<string, schematic.Node>();
    selectedNodes.forEach((n) => m.set(n.key, n));
    return m;
  }, [selectedNodes]);

  const configActions = (
    updates: Iterable<[string, Schematic.ElementConfig]>,
  ): schematic.Action[] =>
    Array.from(updates, ([key, config]) => schematic.setConfig({ key, config }));

  let firstNodeLabel: Schematic.Node.Label.Config | undefined;
  for (const cfg of configByKey.values()) {
    if (!("label" in cfg)) continue;
    firstNodeLabel = cfg.label;
    if (firstNodeLabel != null) break;
  }

  const theme = Theming.use();

  const selection = MultiEdit.selection({
    configs: editableByKey,
    fields: FIELDS,
    onChange: (updates) => dispatch(configActions(updates)),
  });

  const selectionRefs = Array.from(editableByKey).flatMap(([key, cfg]) =>
    selectionColorRefs(key, cfg),
  );

  const handleLayouts = (
    nodeEl: Element,
    nodeElBox: box.Box,
    zoom: number,
  ): Diagram.HandleLayout[] => {
    const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
    return Array.from(handleEls).map((el) => {
      const pos = box.center(box.construct(el));
      const dist = xy.scale(xy.translation(box.topLeft(nodeElBox), pos), 1 / zoom);
      const match = el.className.match(/react-flow__handle-(\w+)/);
      if (match == null)
        throw new Error(`[schematic] - cannot find handle orientation`);
      const orientation = location.outerZ.parse(match[1]);
      return new Diagram.HandleLayout(dist, orientation);
    });
  };

  const getZoom = useCallback(
    () => getViewport({ key: schematicKey }).zoom,
    [schematicKey, getViewport],
  );

  const getLayoutsForAlignment = () => {
    const zoom = getZoom();
    return movable
      .map((nodeKey) => {
        const node = nodesByKey.get(nodeKey);
        if (node == null) return null;
        try {
          const nodeEl = Diagram.selectNode(nodeKey);
          const nodeElBox = box.construct(nodeEl);
          const rect = nodeEl.getBoundingClientRect();
          const actualDims: dimensions.Dimensions = {
            width: rect.width / zoom,
            height: rect.height / zoom,
          };
          const nodeBox = box.construct(node.position, actualDims);
          return new Diagram.NodeLayout(
            nodeKey,
            nodeBox,
            handleLayouts(nodeEl, nodeElBox, zoom),
          );
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  const getLayoutsForDistribution = (): {
    layouts: Diagram.NodeLayout[];
    adjustPosition: (key: string, pos: xy.XY) => xy.XY;
  } => {
    const zoom = getViewport({ key: schematicKey }).zoom;
    const topOffsets = new Map<string, number>();
    const layouts = movable
      .map((nodeKey) => {
        const node = nodesByKey.get(nodeKey);
        if (node == null) return null;
        try {
          const nodeEl = Diagram.selectNode(nodeKey);
          const nodeElBox = box.construct(nodeEl);
          const rect = nodeEl.getBoundingClientRect();

          const gridItems = nodeEl.querySelectorAll(".pluto-grid__item");
          let minTop = rect.top;
          let maxBottom = rect.bottom;
          gridItems.forEach((item) => {
            const itemRect = item.getBoundingClientRect();
            minTop = Math.min(minTop, itemRect.top);
            maxBottom = Math.max(maxBottom, itemRect.bottom);
          });

          const actualDims = {
            width: rect.width / zoom,
            height: (maxBottom - minTop) / zoom,
          };

          const topExtension = (rect.top - minTop) / zoom;
          topOffsets.set(nodeKey, topExtension);
          const adjustedPosition = xy.translate(node.position, {
            x: 0,
            y: -topExtension,
          });

          const nodeBox = box.construct(adjustedPosition, actualDims);
          return new Diagram.NodeLayout(
            nodeKey,
            nodeBox,
            handleLayouts(nodeEl, nodeElBox, zoom),
          );
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);

    const adjustPosition = (key: string, pos: xy.XY): xy.XY => {
      const topOffset = topOffsets.get(key) ?? 0;
      return xy.translate(pos, { x: 0, y: topOffset });
    };
    return { layouts, adjustPosition };
  };

  const nodePositionActions = (layouts: Diagram.NodeLayout[]): schematic.Action[] =>
    layouts.map((n) =>
      schematic.setNodePosition({ key: n.key, position: box.topLeft(n.box) }),
    );

  const applyNodePositions = (layouts: Diagram.NodeLayout[]): void => {
    const actions = nodePositionActions(layouts);
    if (actions.length > 0) dispatch(actions);
  };

  const handleAlignToLocation = (loc: location.Outer): void => {
    applyNodePositions(Diagram.alignNodesToLocation(getLayoutsForAlignment(), loc));
  };

  const handleAlignAlongDirection = (dir: direction.Direction): void => {
    applyNodePositions(Diagram.alignNodesAlongDirection(getLayoutsForAlignment(), dir));
  };

  const handleDistribute = (dir: direction.Direction): void => {
    const { layouts, adjustPosition } = getLayoutsForDistribution();
    const distributed = Diagram.distributeNodes(layouts, dir);
    const adjusted = distributed.map((n) => {
      const pos = adjustPosition(n.key, box.topLeft(n.box));
      return new Diagram.NodeLayout(
        n.key,
        box.construct(pos, box.dims(n.box)),
        n.handles,
      );
    });
    applyNodePositions(adjusted);
  };

  const rotateOrientationActions = (dir: direction.Angular): schematic.Action[] => {
    const updates: [string, Schematic.ElementConfig][] = [];
    editableByKey.forEach((cfg, key) => {
      if (!("orientation" in cfg) || cfg.orientation == null) return;
      updates.push([
        key,
        { ...cfg, orientation: location.rotate(cfg.orientation, dir) },
      ]);
    });
    return configActions(updates);
  };

  const handleRotateIndividual = (dir: direction.Angular): void => {
    const actions = rotateOrientationActions(dir);
    if (actions.length > 0) dispatch(actions);
  };

  const handleRotateGroup = (dir: direction.Angular): void => {
    const actions = [
      ...nodePositionActions(
        Diagram.rotateNodesAroundCenter(getLayoutsForAlignment(), dir),
      ),
      ...rotateOrientationActions(dir),
    ];
    if (actions.length > 0) dispatch(actions);
  };

  const handleLabelProp = <K extends keyof Schematic.Node.Label.Config>(
    key: K,
    value: Schematic.Node.Label.Config[K],
  ): void => {
    const updates: [string, Schematic.ElementConfig][] = [];
    editableByKey.forEach((cfg, elKey) => {
      if (!("label" in cfg) || cfg.label == null) return;
      updates.push([elKey, { ...cfg, label: { ...cfg.label, [key]: value } }]);
    });
    const actions = configActions(updates);
    if (actions.length > 0) dispatch(actions);
  };

  const hasStroke = selection.has("strokeColor");
  const hasFill = selection.has("fillColor");
  const hasText = selection.has("textColor");
  const hasColors = hasStroke || hasFill || hasText || selectionRefs.length > 0;

  return (
    <Form.Sections x>
      <Form.Section title="Arrange">
        <Input.Item label="Align">
          <Flex.Box x>
            <Button.Button
              tooltip="Align symbols vertically"
              onClick={() => handleAlignAlongDirection("x")}
            >
              <Icon.Align.YCenter />
            </Button.Button>
            <Button.Button
              tooltip="Align symbols horizontally"
              onClick={() => handleAlignAlongDirection("y")}
            >
              <Icon.Align.XCenter />
            </Button.Button>
            <Divider.Divider direction="y" />
            <Button.Button
              tooltip="Align symbols left"
              onClick={() => handleAlignToLocation("left")}
            >
              <Icon.Align.Left />
            </Button.Button>
            <Button.Button
              tooltip="Align symbols top"
              onClick={() => handleAlignToLocation("top")}
            >
              <Icon.Align.Top />
            </Button.Button>
            <Button.Button
              tooltip="Align symbols bottom"
              onClick={() => handleAlignToLocation("bottom")}
            >
              <Icon.Align.Bottom />
            </Button.Button>
            <Button.Button
              tooltip="Align symbols right"
              onClick={() => handleAlignToLocation("right")}
            >
              <Icon.Align.Right />
            </Button.Button>
          </Flex.Box>
        </Input.Item>
        {selected.length >= 3 && (
          <Input.Item label="Spacing">
            <Flex.Box x>
              <Button.Button
                tooltip="Distribute symbol spacing horizontally"
                onClick={() => handleDistribute("x")}
              >
                <Icon.Distribute.X />
              </Button.Button>
              <Button.Button
                tooltip="Distribute symbol spacing vertically"
                onClick={() => handleDistribute("y")}
              >
                <Icon.Distribute.Y />
              </Button.Button>
            </Flex.Box>
          </Input.Item>
        )}
        <Input.Item label="Rotate">
          <Flex.Box x>
            <Button.Button
              tooltip="Rotate symbols clockwise"
              onClick={() => handleRotateIndividual("clockwise")}
            >
              <Icon.RotateGroup.CW />
            </Button.Button>
            <Button.Button
              tooltip="Rotate symbols counterclockwise"
              onClick={() => handleRotateIndividual("counterclockwise")}
            >
              <Icon.RotateGroup.CCW />
            </Button.Button>
          </Flex.Box>
        </Input.Item>
        <Input.Item label="Rotate selection">
          <Flex.Box x>
            <Button.Button
              tooltip="Rotate selection clockwise"
              onClick={() => handleRotateGroup("clockwise")}
            >
              <Icon.RotateAroundCenter.CW />
            </Button.Button>
            <Button.Button
              tooltip="Rotate selection counterclockwise"
              onClick={() => handleRotateGroup("counterclockwise")}
            >
              <Icon.RotateAroundCenter.CCW />
            </Button.Button>
          </Flex.Box>
        </Input.Item>
      </Form.Section>
      {hasColors && (
        <MultiEdit.ColorsSection>
          {hasStroke && (
            <MultiEdit.ColorField
              label="Stroke"
              values={selection.colors("strokeColor", (c) =>
                Schematic.colorFallback("strokeColor", c.variant, theme),
              )}
              onChange={(c) => selection.set("strokeColor", c)}
            />
          )}
          {hasFill && (
            <MultiEdit.ColorField
              label="Fill"
              values={selection.colors("fillColor", (c) =>
                Schematic.colorFallback("fillColor", c.variant, theme),
              )}
              onChange={(c) => selection.set("fillColor", c)}
            />
          )}
          {hasText && (
            <MultiEdit.ColorField
              label="Text"
              values={selection.colors("textColor", (c) =>
                Schematic.colorFallback("textColor", c.variant, theme),
              )}
              onChange={(c) => selection.set("textColor", c)}
            />
          )}
          <MultiEdit.SelectionColors
            refs={selectionRefs}
            onChange={selection.setColors}
          />
        </MultiEdit.ColorsSection>
      )}
      {selection.has("label") && (
        <Form.Section title="Label">
          <Input.Item label="Wrap width">
            <Input.Numeric
              value={firstNodeLabel?.maxInlineSize ?? 150}
              onChange={(v) => handleLabelProp("maxInlineSize", v)}
              endContent="px"
            />
          </Input.Item>
          <Input.Item label="Size">
            <Select.Text.Level
              value={firstNodeLabel?.level ?? "p"}
              onChange={(v: text.Level) => handleLabelProp("level", v)}
            />
          </Input.Item>
          <Input.Item label="Alignment">
            <Select.Flex.Alignment
              value={firstNodeLabel?.align ?? "center"}
              onChange={(v: Flex.Alignment) => handleLabelProp("align", v)}
            />
          </Input.Item>
          <Input.Item label="Direction">
            <Direction.Select
              value={firstNodeLabel?.direction ?? "x"}
              onChange={(v: direction.Direction) => handleLabelProp("direction", v)}
              yDirection="down"
            />
          </Input.Item>
          <Input.Item label="Location">
            <Schematic.Node.Orientation.Select
              value={{ inner: "top", outer: firstNodeLabel?.orientation ?? "top" }}
              onChange={(v) =>
                v.outer !== "center" && handleLabelProp("orientation", v.outer)
              }
              hideInner
            />
          </Input.Item>
        </Form.Section>
      )}
      {selection.has("scale") && (
        <Form.Section title="Symbol size">
          <Input.Item label="Scale" align="start" padHelpText={false}>
            <Input.Numeric
              bounds={SCALE_BOUNDS}
              endContent="%"
              value={Math.round((selection.first("scale") ?? 1) * 100)}
              onChange={(v) => selection.set("scale", parseFloat((v / 100).toFixed(2)))}
            />
          </Input.Item>
        </Form.Section>
      )}
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
      {selection.has("control") && (
        <Form.Section title="Control">
          <Input.Item label="Control chip" align="start" padHelpText={false}>
            <Input.Switch
              value={selection.first("control")?.hidden !== true}
              onChange={(v) =>
                selection.update("control", (c) =>
                  schematic.controlStateConfigZ.parse({ ...c, hidden: !v }),
                )
              }
            />
          </Input.Item>
        </Form.Section>
      )}
    </Form.Sections>
  );
};
