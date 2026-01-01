// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { type CSSProperties, type FC } from "react";

import { Handle } from "@/arc/handle";
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
    style={{
      padding: "0.5rem 1rem",
    }}
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

const BORDER_STYLES: Record<location.X, React.CSSProperties> = {
  left: {
    borderTopLeftRadius: "1rem",
    borderBottomLeftRadius: "1rem",
    marginRight: "-1px",
    zIndex: -1,
  },
  right: {
    borderTopRightRadius: "1rem",
    borderBottomRightRadius: "1rem",
    marginLeft: "-1px",
    zIndex: -1,
  },
};

const HANDLE_STYLES: Record<location.X, React.CSSProperties> = {
  left: {
    left: "-0.5rem",
  },
  right: {
    right: "-0.5rem",
  },
};

const createHandles = (
  location: location.X,
  HandleC: typeof Handle.Source | typeof Handle.Sink,
): FC<HandlesProps> => {
  const C = ({ handles: inputs = [], center = false }: HandlesProps) => {
    if (inputs.length === 0) return null;
    const adjustedStyle: CSSProperties = {
      ...BORDER_STYLES[location],
      padding: "0.5rem",
      height: "fit-content",
    };
    if (center) {
      adjustedStyle.marginTop = "auto";
      adjustedStyle.marginBottom = "auto";
    }
    return (
      <Flex.Box
        y
        align="center"
        gap={0.5}
        background={2}
        bordered
        borderColor={6}
        style={adjustedStyle}
        justify="around"
      >
        {inputs.map((input) => {
          const Icon = input.Icon;
          return (
            <div
              key={input.key}
              style={{
                position: "relative",
              }}
            >
              <Icon style={{ width: "2.5rem", height: "2.5rem" }} />
              <HandleC
                key={input.key}
                id={input.key}
                location={location}
                style={{
                  position: "absolute",
                  top: "50%",
                  ...HANDLE_STYLES[location],
                }}
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
  <Minimal sources={sources} sinks={sinks} style={{ padding: "1rem" }} scale={scale}>
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
  const adjustedStyle: CSSProperties = { ...style };
  if (sinkHandleCount === 0 || centerSinks) adjustedStyle.borderTopLeftRadius = "1rem";
  if (sinkHandleCount < 2 || centerSinks) adjustedStyle.borderBottomLeftRadius = "1rem";
  if (sourceHandleCount === 0 || centerSources)
    adjustedStyle.borderTopRightRadius = "1rem";
  if (sourceHandleCount < 2 || centerSources)
    adjustedStyle.borderBottomRightRadius = "1rem";

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
