// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import {
  Align,
  Button,
  Diagram,
  Form,
  Input,
  Schematic,
  Status,
} from "@synnaxlabs/pluto";
import { Color } from "@synnaxlabs/pluto/color";
import { box, deep, location, xy } from "@synnaxlabs/x";
import { memo, type ReactElement } from "react";
import { useDispatch } from "react-redux";

import {
  type ElementInfo,
  type NodeElementInfo,
  useSelectSelectedElementsProps,
  useSelectViewport,
} from "@/schematic/selectors";
import { setElementProps, setNodePositions } from "@/schematic/slice";

export interface PropertiesProps {
  layoutKey: string;
}

export const PropertiesControls = memo(
  ({ layoutKey }: PropertiesProps): ReactElement => {
    const elements = useSelectSelectedElementsProps(layoutKey);
    const viewport = useSelectViewport(layoutKey);

    const dispatch = useDispatch();

    const handleChange = (key: string, props: any): void => {
      dispatch(setElementProps({ layoutKey, key, props }));
    };

    if (elements.length === 0)
      return (
        <Status.Text.Centered variant="disabled" hideIcon>
          Select a Schematic element to configure its properties.
        </Status.Text.Centered>
      );

    if (elements.length > 1) {
      const groups: Record<string, ElementInfo[]> = {};
      elements.forEach((e) => {
        let color: Color.Color | null = null;
        if (e.type === "edge") color = new Color.Color(e.edge.color);
        else if (e.props.color != null) color = new Color.Color(e.props.color);
        if (color === null) return;
        const hex = color.hex;
        if (!(hex in groups)) groups[hex] = [];
        groups[hex].push(e);
      });

      const layouts = elements
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
                1 / viewport.zoom,
              );
              const match = el.className.match(/react-flow__handle-(\w+)/);
              if (match == null)
                throw new Error(`[schematic] - cannot find handle orientation`);
              const orientation = location.construct(match[1]) as location.Outer;
              return new Diagram.HandleLayout(dist, orientation);
            });
            return new Diagram.NodeLayout(el.key, nodeBox, handles);
          } catch (e) {
            console.error(e);
          }
          return null;
        })
        .filter((el) => el !== null);

      return (
        <Align.Space align="start" direction="x" style={{ padding: "2rem" }}>
          <Input.Item label="Selection Colors" align="start">
            <Align.Space direction="y">
              {Object.entries(groups).map(([hex, elements]) => (
                <Color.Swatch
                  key={elements[0].key}
                  value={hex}
                  onChange={(color: Color.Color) => {
                    elements.forEach((e) => handleChange(e.key, { color: color.hex }));
                  }}
                />
              ))}
            </Align.Space>
          </Input.Item>
          <Input.Item label="Align">
            <Align.Space direction="x">
              <Button.Icon
                tooltip="Align nodes vertically"
                onClick={() => {
                  const newPositions = Diagram.alignNodes(layouts, "x");
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
              </Button.Icon>
              <Button.Icon
                tooltip="Align nodes horizontally"
                onClick={() => {
                  const newPositions = Diagram.alignNodes(layouts, "y");
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
              </Button.Icon>
            </Align.Space>
          </Input.Item>
        </Align.Space>
      );
    }

    const selected = elements[0];

    if (selected.type === "edge")
      return <EdgeProperties edge={selected} onChange={handleChange} />;
    return (
      <IndividualProperties
        key={selected.key}
        element={selected}
        onChange={handleChange}
      />
    );
  },
);
PropertiesControls.displayName = "PropertiesControls";

const IndividualProperties = ({
  element: selected,
  onChange,
}: {
  element: NodeElementInfo;
  onChange: (key: string, props: any) => void;
}): ReactElement => {
  const C = Schematic.SYMBOLS[selected.props.key];

  const formMethods = Form.use({
    values: deep.copy(selected.props),
    sync: true,
    onChange: ({ values }) => onChange(selected.key, values),
  });

  return (
    <Align.Space style={{ height: "100%" }} direction="y">
      <Form.Form {...formMethods}>
        <C.Form {...formMethods} key={selected.key} />
      </Form.Form>
    </Align.Space>
  );
};

interface EdgePropertiesProps {
  edge: ElementInfo;
  onChange: (key: string, props: any) => void;
}

const EdgeProperties = ({ edge, onChange }: EdgePropertiesProps): ReactElement => {
  if (edge.type !== "edge") return <></>;
  return (
    <Align.Space style={{ padding: "2rem" }} align="start" direction="x">
      <Input.Item label="Color" align="start">
        <Color.Swatch
          value={edge.edge.color ?? Color.ZERO}
          onChange={(color: Color.Color) => {
            onChange(edge.key, { color: color.hex });
          }}
        />
      </Input.Item>
      <Input.Item label="Type" align="start">
        <Diagram.SelectPathType
          value={edge.edge.variant as Diagram.PathType}
          onChange={(variant: Diagram.PathType) => onChange(edge.key, { variant })}
        />
      </Input.Item>
    </Align.Space>
  );
};
