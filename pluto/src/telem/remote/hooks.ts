// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type telem } from "@/telem/core";
import {
  NumericSource,
  type NumericSourceProps as RemoteTelemNumericProps,
} from "@/telem/remote/aether/numeric";
import {
  type DynamicSeriesSourceProps,
  SeriesSource,
  type SeriesSourceProps,
  DynamicSeriesSource,
} from "@/telem/remote/aether/series";

export const seriesSource = (props: SeriesSourceProps): telem.SeriesSourceSpec => {
  return {
    type: SeriesSource.TYPE,
    props,
    variant: "series-source",
  };
};

export const dynamicSeriesSource = (
  props: DynamicSeriesSourceProps,
): telem.SeriesSourceSpec => {
  return {
    type: DynamicSeriesSource.TYPE,
    props,
    variant: "series-source",
  };
};

export const useNumericSource = (
  props: Omit<RemoteTelemNumericProps, "units">,
): telem.NumericSourceSpec => {
  return {
    type: NumericSource.TYPE,
    props,
    variant: "numeric-source",
  };
};

export const useNumericStringSource = (
  props: RemoteTelemNumericProps,
): telem.StringSourceSpec => {
  return {
    type: NumericSource.TYPE,
    props,
    variant: "string-source",
  };
};
