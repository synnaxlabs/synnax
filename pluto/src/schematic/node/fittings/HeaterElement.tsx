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

export interface HeaterElementProps
  extends Primitive.DivProps, Primitive.SVGBasedProps {}

const DIMENSIONS = { width: 21, height: 36 };

export const HeaterElement = ({
  className,
  orientation = "left",
  color,
  scale,
  ...rest
}: HeaterElementProps): ReactElement => (
  <Primitive.Div className={CSS.cx(CSS.B("heater-element"), className)} {...rest}>
    <Handle.Boundary orientation={orientation}>
      <Handle.Handle
        location="top"
        orientation={orientation}
        left={11.9}
        top={5.55555556}
        id="3"
      />
      <Handle.Handle
        location="bottom"
        orientation={orientation}
        left={11.9}
        top={94} // not sure why but this rounding looks better
        id="4"
      />
    </Handle.Boundary>
    <Primitive.SVG
      dimensions={DIMENSIONS}
      color={color}
      orientation={orientation}
      scale={scale}
    >
      <Primitive.Path d="M2.5 34.0005H14.5" strokeLinecap="round" />
      <Primitive.Path d="M2.5 2.00049L14.5 2.00049" strokeLinecap="round" />
      <Primitive.Path d="M14.3793 10.0013C14.9046 10.0013 15.4248 10.1048 15.9101 10.3058C16.3954 10.5068 16.8363 10.8015 17.2078 11.1729C17.5792 11.5443 17.8738 11.9853 18.0749 12.4706C18.2759 12.9559 18.3793 13.4761 18.3793 14.0013C18.3793 14.5266 18.2759 15.0468 18.0749 15.5321C17.8738 16.0174 17.5792 16.4583 17.2078 16.8298C16.8363 17.2012 16.3954 17.4958 15.9101 17.6969C15.4248 17.8979 14.9046 18.0013 14.3793 18.0013" />
      <Primitive.Path d="M14.5 2.00049C15.0253 2.00049 15.5454 2.10395 16.0307 2.30497C16.516 2.50599 16.957 2.80063 17.3284 3.17206C17.6999 3.5435 17.9945 3.98445 18.1955 4.46975C18.3965 4.95506 18.5 5.4752 18.5 6.00049C18.5 6.52578 18.3965 7.04592 18.1955 7.53122C17.9945 8.01652 17.6999 8.45748 17.3284 8.82892C16.957 9.20035 16.516 9.49499 16.0307 9.69601C15.5454 9.89703 15.0253 10.0005 14.5 10.0005" />
      <Primitive.Path d="M14.5 18.0005C15.0253 18.0005 15.5454 18.104 16.0307 18.305C16.516 18.506 16.957 18.8006 17.3284 19.1721C17.6999 19.5435 17.9945 19.9845 18.1955 20.4698C18.3965 20.9551 18.5 21.4752 18.5 22.0005C18.5 22.5258 18.3965 23.0459 18.1955 23.5312C17.9945 24.0165 17.6999 24.4575 17.3284 24.8289C16.957 25.2003 16.516 25.495 16.0307 25.696C15.5454 25.897 15.0253 26.0005 14.5 26.0005" />
      <Primitive.Path d="M14.6206 25.9995C15.1459 25.9995 15.6661 26.103 16.1514 26.304C16.6367 26.505 17.0776 26.7997 17.4491 27.1711C17.8205 27.5425 18.1151 27.9835 18.3162 28.4688C18.5172 28.9541 18.6206 29.4742 18.6206 29.9995C18.6206 30.5248 18.5172 31.0449 18.3162 31.5302C18.1151 32.0155 17.8205 32.4565 17.4491 32.8279C17.0776 33.1994 16.6367 33.494 16.1514 33.695C15.6661 33.896 15.1459 33.9995 14.6206 33.9995" />
    </Primitive.SVG>
  </Primitive.Div>
);
