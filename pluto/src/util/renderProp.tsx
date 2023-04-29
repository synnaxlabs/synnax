// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type RenderProp<P extends Record<string, any>> = (props: P) => JSX.Element | null;

/**
 * Component prop takes in a component and turns it into a render prop.
 */
export const componentRenderProp = <P extends Record<string, any>>(Component: React.ComponentType<P>): RenderProp<P> =>
  (props) => <Component {...props} />

