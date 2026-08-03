// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/arc/graph/node/Base.css";

import { type location } from "@synnaxlabs/x";
import { type CSSProperties, type FC, useMemo } from "react";

import { Handle } from "@/arc/graph/handle";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { type Icon } from "@/icon";
import { Text } from "@/text";

export interface HandleSpec {
  key: string;
  Icon: Icon.FC;
}

export interface BaseProps extends MinimalProps {
  type?: string;
  Icon?: Icon.ReactElement;
  color: string;
  textColor: string;
}
export interface TypeTextProps {
  type: string;
  icon: Icon.ReactElement;
  color: string;
  textColor: string;
}

export const TypeText = ({ type, icon, color, textColor }: TypeTextProps) => (
  <Text.Text
    level="small"
    gap="tiny"
    weight={500}
    bordered
    overflow="nowrap"
    variant="code"
    rounded={0.5}
    background={color}
    borderColor={textColor}
    className={CSS.BE("arc", "type-text")}
    color={textColor}
  >
    {icon}
    {type}
  </Text.Text>
);

interface HandlesProps {
  center?: boolean;
  handles?: HandleSpec[];
}

const HANDLE_POSITION_STYLE: Record<location.X, CSSProperties> = {
  left: { position: "absolute", top: "50%", left: "-0.5rem" },
  right: { position: "absolute", top: "50%", right: "-0.5rem" },
};

const createHandles = (
  location: location.X,
  HandleC: typeof Handle.Source | typeof Handle.Sink,
): FC<HandlesProps> => {
  const C = ({ handles: inputs = [], center = false }: HandlesProps) => {
    if (inputs.length === 0) return null;
    return (
      <Flex.Box
        y
        align="center"
        gap={0.5}
        background={2}
        bordered
        borderColor={6}
        className={CSS(
          CSS.BE("arc", "handles"),
          CSS.BEM("arc", "handles", location),
          center && CSS.BEM("arc", "handles", "center"),
        )}
        justify="around"
      >
        {inputs.map((input) => {
          const Icon = input.Icon;
          return (
            <div key={input.key} className={CSS.BE("arc", "handle-wrapper")}>
              <Icon className={CSS.BE("arc", "handle-icon")} />
              <HandleC
                key={input.key}
                id={input.key}
                location={location}
                style={HANDLE_POSITION_STYLE[location]}
              />
            </div>
          );
        })}
      </Flex.Box>
    );
  };
  C.displayName = `Handles(${location})`;
  return C;
};

const SinkHandles = createHandles("left", Handle.Sink);
const SourceHandles = createHandles("right", Handle.Source);

const BASE_NODE_STYLE: CSSProperties = { padding: "1rem" };

export const Base = ({
  type,
  Icon: icon,
  sources,
  sinks,
  color,
  textColor,
  children,
  scale,
}: BaseProps) => (
  <Minimal sources={sources} sinks={sinks} style={BASE_NODE_STYLE} scale={scale}>
    <Configuration
      type={type}
      icon={icon}
      color={color}
      textColor={textColor}
      sourceHandleCount={sources?.length ?? 0}
      sinkHandleCount={sinks?.length ?? 0}
    >
      {children}
    </Configuration>
  </Minimal>
);

interface MinimalProps {
  sources?: HandleSpec[];
  centerSources?: boolean;
  sinks?: HandleSpec[];
  centerSinks?: boolean;
  children: React.ReactNode;
  style?: CSSProperties;
  scale?: number;
}

export const Minimal = ({
  sources,
  sinks,
  children,
  style,
  centerSources,
  centerSinks,
}: MinimalProps) => {
  const sinkHandleCount = sinks?.length ?? 0;
  const sourceHandleCount = sources?.length ?? 0;
  const adjustedStyle = useMemo(() => {
    const s: CSSProperties = { ...style };
    if (sinkHandleCount === 0 || centerSinks) s.borderTopLeftRadius = "1rem";
    if (sinkHandleCount < 2 || centerSinks) s.borderBottomLeftRadius = "1rem";
    if (sourceHandleCount === 0 || centerSources) s.borderTopRightRadius = "1rem";
    if (sourceHandleCount < 2 || centerSources) s.borderBottomRightRadius = "1rem";
    return s;
  }, [style, sinkHandleCount, sourceHandleCount, centerSinks, centerSources]);

  return (
    <Flex.Box x empty className={CSS.BE("arc", "stage")}>
      <SinkHandles handles={sinks} center={centerSinks} />
      <Flex.Box
        y
        className={CSS.BE("arc", "stage", "body")}
        background={0}
        bordered
        borderColor={6}
        gap="small"
        justify="center"
        style={adjustedStyle}
        align="start"
      >
        {children}
      </Flex.Box>
      <SourceHandles handles={sources} center={centerSources} />
    </Flex.Box>
  );
};

interface ConfigurationProps {
  children: React.ReactNode;
  type?: string;
  icon?: Icon.ReactElement;
  color: string;
  textColor: string;
  sourceHandleCount: number;
  sinkHandleCount: number;
}

const Configuration = ({
  children,
  type,
  icon,
  color,
  textColor,
}: ConfigurationProps) => (
  <>
    {type != null && icon != null && (
      <TypeText type={type} icon={icon} color={color} textColor={textColor} />
    )}
    {children}
  </>
);
