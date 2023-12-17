// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  Status,
  PID,
  Color,
  Input,
  Align,
  PIDSymbols,
  Button,
} from "@synnaxlabs/pluto";
import { box, location, xy } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import {
  type ElementInfo,
  useSelectSelectedElementsProps,
  type NodeElementInfo,
  useSelectViewport,
} from "@/pid/selectors";
import { setElementProps, setNodePositions, setNodes } from "@/pid/slice";

import "@/pid/toolbar/Properties.css";

export interface PropertiesProps {
  layoutKey: string;
}

export const PropertiesControls = ({ layoutKey }: PropertiesProps): ReactElement => {
  const elements = useSelectSelectedElementsProps(layoutKey);
  const viewport = useSelectViewport(layoutKey);

  const dispatch = useDispatch();

  const handleChange = (key: string, props: any): void => {
    dispatch(setElementProps({ layoutKey, key, props }));
  };

  if (elements.length === 0)
    return (
      <Status.Text.Centered variant="disabled" hideIcon>
        Select a PID element to configure its properties.
      </Status.Text.Centered>
    );

  if (elements.length > 1) {
    const groups: Record<string, ElementInfo[]> = {};
    elements.forEach((e) => {
      let color: Color.Color | null = null;
      if (e.type === "edge") color = new Color.Color(e.edge.color ?? Color.ZERO);
      else if ("color" in e.props) color = new Color.Color(e.props.color);
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
          const nodeEl = PID.selectNode(el.key);
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
              throw new Error(`[pid] - cannot find handle orientation`);
            const orientation = location.construct(match[1]) as location.Outer;
            return new PID.HandleLayout(dist, orientation);
          });
          return new PID.NodeLayout(el.key, nodeBox, handles);
        } catch (e) {
          console.log(e);
        }
      })
      .filter((el) => el !== null) as PID.NodeLayout[];

    return (
      <Align.Space
        className={CSS.B("pid-properties-multi")}
        align="start"
        direction="x"
      >
        <Input.Item label="Selection Colors" align="start">
          <Align.Space direction="y">
            {Object.entries(groups).map(([hex, elements]) => {
              return (
                <Color.Swatch
                  key={elements[0].key}
                  value={hex}
                  onChange={(color) => {
                    elements.forEach((e) => {
                      handleChange(e.key, { color: color.hex });
                    });
                  }}
                />
              );
            })}
          </Align.Space>
        </Input.Item>
        <Input.Item label="Align">
          <Align.Space direction="x">
            <Button.Icon
              tooltip="Align nodes vertically"
              onClick={() => {
                const newPositions = PID.alignNodes(layouts, "x");
                dispatch(
                  setNodePositions({
                    layoutKey,
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
                const newPositions = PID.alignNodes(layouts, "y");
                dispatch(
                  setNodePositions({
                    layoutKey,
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

  if (selected.type === "edge") {
    return (
      <Color.Swatch
        value={selected.edge.color ?? Color.ZERO}
        onChange={(color) => {
          handleChange(selected.key, { color: color.hex });
        }}
      />
    );
  }

  const C = PIDSymbols.registry[selected.props.variant];

  return (
    <Align.Space className={CSS.B("pid-properties")} size="small">
      <C.Form
        key={selected.key}
        value={selected.props}
        onChange={(props) => handleChange(selected.key, props)}
      />
    </Align.Space>
  );
};

export const fromCSSTransform = (transform: string): XY => {
  const [x, y] = transform
    .replace("translate(", "")
    .replace(")", "")
    .split(",")
    .map((s) => parseFloat(s));
  return { x, y };
};

export const alignItems = (elements: ElementInfo[]): void => {};

export const alignItemsLeg = (elements: ElementInfo[]): PID.Node[] => {
  const nodes = elements.filter((e) => e.type === "node") as NodeElementInfo[];
  const edges = elements.filter((e) => e.type === "edge");
  const htmlElements = nodes
    .map((n) => document.querySelector(`[data-id="${n.key}"]`))
    .filter((e) => e !== null) as HTMLElement[];

  const handlePositions = htmlElements.map((e, i) => {
    const node = nodes[i];
    console.log(nodes[i].node.position, box.top(box.construct(e)));
    const right = e.querySelector(".react-flow__handle-right");
    // const left = e.querySelector(".react-flow__handle-left");
    const els = [right].filter((e) => e !== null) as HTMLElement[];
    // reduce average box.center
    const avg =
      els.reduce((acc, handle) => acc + box.center(box.construct(handle)).y, 0) /
        els.length -
      box.top(box.construct(e));
    return avg + node.node.position.y;
  });
  const overallAverage =
    handlePositions.reduce((acc, y, i) => acc + y) / htmlElements.length;
  return nodes.map((n, i) => {
    const delta = handlePositions[i] - n.node.position.y;
    const next = {
      ...n.node,
      position: {
        ...n.node.position,
        y: overallAverage - delta,
      },
    };
    return next;
  });
};
