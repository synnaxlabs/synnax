// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Handle } from "@/schematic/node/common/handle";
import { Primitive } from "@/schematic/node/common/primitive";
import { Toggle } from "@/schematic/node/common/toggle";
export interface Props extends Toggle.ButtonProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 89, height: 76 };

export const SpringLoadedRelief = ({
  className,
  orientation = "left",
  color: colorVal,
  scale,
  enabled = false,
  ...rest
}: Props): ReactElement => (
    <Toggle.Button
      {...rest}
      orientation={orientation}
      className={CSS(CSS.B("spring-loaded-relief-valve"), className)}
      enabled={enabled}
    >
      <Handle.Boundary orientation={orientation}>
        <Handle.Handle
          location="left"
          orientation={orientation}
          left={2.1373}
          top={69.0785}
          id="1"
        />
        <Handle.Handle
          location="right"
          orientation={orientation}
          left={96.7416}
          top={72.368}
          id="2"
        />
      </Handle.Boundary>
      <Primitive.SVG
        dimensions={DIMENSIONS}
        color={colorVal}
        orientation={orientation}
        scale={scale}
      >
        <Primitive.Path
          d="M46.3625 54.1079C45.6312 54.4779 45.6311 55.5223 46.3624 55.8924L81.7435 73.7989C83.7389 74.8088 86.098 73.3588 86.0982 71.1224L86.1002 38.883C86.1003 36.6465 83.7414 35.1962 81.7458 36.2059L46.3625 54.1079Z"
          stroke="var(--pluto-symbol-display)"
        />
        <Primitive.Path
          d="M71 38.0014V72.0014"
          stroke="var(--pluto-symbol-display)"
          strokeWidth={4}
          strokeLinecap="round"
        />
        <Primitive.Path
          d="M41.6389 55.8923C42.3702 55.5222 42.3702 54.4778 41.6389 54.1077L6.2567 36.2035C4.26118 35.1937 1.90217 36.6438 1.90217 38.8803L1.90217 71.1197C1.90217 73.3562 4.26119 74.8063 6.2567 73.7965L41.6389 55.8923Z"
          stroke="var(--pluto-symbol-display)"
        />
        <Primitive.Circle cx="44" cy="55" r="4" fill="var(--pluto-symbol-display)" />
        <Primitive.Path
          d="M45 4C45 3.44772 44.5523 3 44 3C43.4477 3 43 3.44772 43 4H45ZM37.5777 24.7889L38.0249 23.8944L37.5777 24.7889ZM37.5777 21.2111L38.0249 22.1056L37.5777 21.2111ZM50.4223 31.2111L49.9751 32.1056L50.4223 31.2111ZM37.5777 44.7889L37.1305 45.6833L37.5777 44.7889ZM37.5777 41.2111L37.1305 40.3167L37.5777 41.2111ZM42.8944 47.4472L42.4472 48.3416L42.8944 47.4472ZM45 52V49.2361H43V52H45ZM43.3416 46.5528L38.0249 43.8944L37.1305 45.6833L42.4472 48.3416L43.3416 46.5528ZM38.0249 42.1056L50.8695 35.6833L49.9751 33.8944L37.1305 40.3167L38.0249 42.1056ZM50.8695 30.3167L38.0249 23.8944L37.1305 25.6833L49.9751 32.1056L50.8695 30.3167ZM38.0249 22.1056L50.8695 15.6833L49.9751 13.8944L37.1305 20.3167L38.0249 22.1056ZM50.8695 10.3167L45.5528 7.65836L44.6584 9.44721L49.9751 12.1056L50.8695 10.3167ZM45 6.76393V4H43V6.76393H45ZM45.5528 7.65836C45.214 7.48897 45 7.1427 45 6.76393H43C43 7.90025 43.642 8.93904 44.6584 9.44721L45.5528 7.65836ZM50.8695 15.6833C53.0806 14.5777 53.0807 11.4223 50.8695 10.3167L49.9751 12.1056C50.7121 12.4741 50.7121 13.5259 49.9751 13.8944L50.8695 15.6833ZM38.0249 23.8944C37.2879 23.5259 37.2879 22.4741 38.0249 22.1056L37.1305 20.3167C34.9193 21.4223 34.9194 24.5777 37.1305 25.6833L38.0249 23.8944ZM50.8695 35.6833C53.0806 34.5777 53.0807 31.4223 50.8695 30.3167L49.9751 32.1056C50.7121 32.4741 50.7121 33.5259 49.9751 33.8944L50.8695 35.6833ZM38.0249 43.8944C37.2879 43.5259 37.2879 42.4741 38.0249 42.1056L37.1305 40.3167C34.9193 41.4223 34.9194 44.5777 37.1305 45.6833L38.0249 43.8944ZM45 49.2361C45 48.0998 44.358 47.061 43.3416 46.5528L42.4472 48.3416C42.786 48.511 43 48.8573 43 49.2361H45Z"
          fill="var(--pluto-symbol-display)"
          strokeWidth={1}
        />
      </Primitive.SVG>
    </Toggle.Button>
  );
