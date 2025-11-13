// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import {
  Button,
  Color,
  Diagram,
  Flex,
  Form,
  Icon,
  Input,
  Schematic,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { box, color, deep, location, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, type ReactNode } from "react";
import { useDispatch, useStore } from "react-redux";

import { Layout } from "@/layout";
import {
  type ElementInfo,
  selectViewport,
  useSelectRequiredEdge,
  useSelectRequiredNodeProps,
  useSelectRequired,
  useSelectSelectedElementDigests,
  useSelectSelectedElementsProps,
} from "@/schematic/selectors";
import { setElementProps, setLegend, setNodePositions } from "@/schematic/slice";
import { createEditLayout } from "@/schematic/symbols/edit/Edit";
import { type EdgeProps, type NodeProps } from "@/schematic/types";
import { type nodePropsZ } from "@/schematic/types/v0";
import { type RootState } from "@/store";

export interface PropertiesProps {
  layoutKey: string;
}

export const PropertiesControls = memo(
  ({ layoutKey }: PropertiesProps): ReactElement => {
    const digests = useSelectSelectedElementDigests(layoutKey);
    const selected = digests.length === 1 ? digests[0] : null;

    return (
      <Flex.Box y style={{ height: "100%", overflow: "auto" }}>
        <SchematicProperties layoutKey={layoutKey} />
        {digests.length === 0 ? (
          <Text.Text status="disabled" center style={{ padding: "2rem" }}>
            Select a Schematic element to configure its properties.
          </Text.Text>
        ) : digests.length > 1 ? (
          <MultiElementProperties layoutKey={layoutKey} />
        ) : selected != null && selected.type === "edge" ? (
          <EdgeProperties layoutKey={layoutKey} edgeKey={selected.key} />
        ) : selected != null ? (
          <IndividualProperties
            key={selected.key}
            layoutKey={layoutKey}
            nodeKey={selected.key}
          />
        ) : null}
      </Flex.Box>
    );
  },
);
PropertiesControls.displayName = "PropertiesControls";

const SchematicProperties = ({ layoutKey }: PropertiesProps): ReactElement => {
  const schematic = useSelectRequired(layoutKey);
  const dispatch = useDispatch();

  const handleLegendVisibilityChange = (value: boolean): void => {
    dispatch(setLegend({ key: layoutKey, legend: { visible: value } }));
  };

  return (
    <Flex.Box x style={{ padding: "2rem", borderBottom: "var(--pluto-border)" }}>
      <Input.Item label="Show Control State Legend">
        <Input.Switch
          value={schematic.legend.visible}
          onChange={handleLegendVisibilityChange}
        />
      </Input.Item>
    </Flex.Box>
  );
};

interface IndividualPropertiesProps {
  layoutKey: string;
  nodeKey: string;
}

const IndividualProperties = ({
  layoutKey,
  nodeKey,
}: IndividualPropertiesProps): ReactElement | null => {
  const props = useSelectRequiredNodeProps(layoutKey, nodeKey);
  const C = Schematic.Symbol.REGISTRY[props.key];
  const dispatch = useDispatch();

  const onChange = (key: string, props: NodeProps): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };

  const formMethods = Form.use<typeof nodePropsZ>({
    values: deep.copy(props),
    sync: true,
    onChange: ({ values }) => onChange(nodeKey, deep.copy(values)),
  });
  const specKey = Form.useFieldValue<string, string, typeof nodePropsZ>("specKey", {
    ctx: formMethods,
    optional: true,
  });
  const isRemote = schematic.symbol.keyZ.safeParse(specKey).success;
  let actions: ReactNode = null;
  const placeLayout = Layout.usePlacer();
  if (isRemote && specKey != null)
    actions = (
      <Button.Button
        variant="filled"
        size="tiny"
        style={{ marginRight: "1rem" }}
        onClick={() => placeLayout(createEditLayout({ args: { key: specKey } }))}
      >
        <Icon.Edit />
      </Button.Button>
    );

  return (
    <Flex.Box style={{ height: "100%" }} y>
      <Form.Form<typeof nodePropsZ> {...formMethods}>
        <C.Form {...formMethods} key={nodeKey} actions={actions} />
      </Form.Form>
    </Flex.Box>
  );
};

interface EdgePropertiesProps {
  layoutKey: string;
  edgeKey: string;
}

const EdgeProperties = ({
  layoutKey,
  edgeKey,
}: EdgePropertiesProps): ReactElement | null => {
  const edge = useSelectRequiredEdge(layoutKey, edgeKey);
  const dispatch = useDispatch();
  const onChange = (key: string, props: Partial<EdgeProps>): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };
  return (
    <Flex.Box style={{ padding: "2rem" }} align="start" x>
      <Input.Item label="Color" align="start">
        <Color.Swatch
          value={(edge.data?.color ?? color.ZERO) as color.Crude}
          onChange={(v: color.Color) => {
            onChange(edge.key, { color: color.hex(v) });
          }}
        />
      </Input.Item>
      <Input.Item label="Type" align="start">
        <Schematic.SelectEdgeType
          value={edge.data?.variant as Schematic.EdgeType}
          onChange={(variant: Schematic.EdgeType) => onChange(edge.key, { variant })}
        />
      </Input.Item>
    </Flex.Box>
  );
};

interface MultiElementPropertiesProps {
  layoutKey: string;
}

const MultiElementProperties = ({
  layoutKey,
}: MultiElementPropertiesProps): ReactElement => {
  const handleError = Status.useErrorHandler();
  const elements = useSelectSelectedElementsProps(layoutKey);
  const dispatch = useDispatch();
  const onChange = (key: string, props: Partial<NodeProps>): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };

  const colorGroups: Record<string, ElementInfo[]> = {};
  elements.forEach((e) => {
    let colorVal: color.Color | null = null;
    if (e.type === "edge") colorVal = color.colorZ.parse(e.edge.data?.color);
    else if (e.props.color != null) colorVal = color.construct(e.props.color);
    if (colorVal === null) return;
    const hex = color.hex(colorVal);
    if (!(hex in colorGroups)) colorGroups[hex] = [];
    colorGroups[hex].push(e);
  });

  const store = useStore<RootState>();

  const getLayouts = () => {
    const viewport = selectViewport(store.getState(), layoutKey);
    return elements
      .map((el) => {
        if (el.type !== "node") return null;
        // grab all child elements with the class 'react-flow__handle'
        try {
          const nodeEl = Diagram.selectNode(el.key);
          const nodeBox = box.construct(
            el.node.position,
            box.dims(box.construct(nodeEl)),
          );
          const handleEls = nodeEl.getElementsByClassName("react-flow__handle");
          const nodeElBox = box.construct(nodeEl);
          const handles = Array.from(handleEls).map((el) => {
            const pos = box.center(box.construct(el));
            const dist = xy.scale(
              xy.translation(box.topLeft(nodeElBox), pos),
              1 / (viewport?.zoom ?? 1),
            );
            const match = el.className.match(/react-flow__handle-(\w+)/);
            if (match == null)
              throw new Error(`[schematic] - cannot find handle orientation`);
            const orientation = location.construct(match[1]) as location.Outer;
            return new Diagram.HandleLayout(dist, orientation);
          });
          return new Diagram.NodeLayout(el.key, nodeBox, handles);
        } catch (e) {
          handleError(e, "failed to calculate schematic node layout");
        }
        return null;
      })
      .filter((el) => el !== null);
  };

  return (
    <Flex.Box align="start" x style={{ padding: "2rem" }} gap="large">
      <Input.Item label="Selection Colors" align="start">
        <Flex.Box x>
          {Object.entries(colorGroups).map(([hex, elements]) => (
            <Color.Swatch
              key={elements[0].key}
              value={hex}
              onChange={(v: color.Color) => {
                elements.forEach((e) => onChange(e.key, { color: color.hex(v) }));
              }}
            />
          ))}
        </Flex.Box>
      </Input.Item>
      <Input.Item label="Align">
        <Flex.Box x>
          <Button.Button
            tooltip="Align nodes vertically"
            onClick={() => {
              const newPositions = Diagram.alignNodes(getLayouts(), "x");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.YCenter />
          </Button.Button>
          <Button.Button
            tooltip="Align nodes horizontally"
            onClick={() => {
              const newPositions = Diagram.alignNodes(getLayouts(), "y");
              dispatch(
                setNodePositions({
                  key: layoutKey,
                  positions: Object.fromEntries(
                    newPositions.map((n) => [n.key, box.topLeft(n.box)]),
                  ),
                }),
              );
            }}
          >
            <Icon.Align.XCenter />
          </Button.Button>
        </Flex.Box>
      </Input.Item>
    </Flex.Box>
  );
};
