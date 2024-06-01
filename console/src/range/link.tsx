// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Link } from "@/link";
import { setActive } from "@/range/slice";

export const linkHandler: Link.Handler = ({
  resource,
  resourceKey,
  client,
  dispatch,
}) => {
  if (resource != "range") return false;
  client.ranges
    .retrieve(resourceKey)
    .then((range) => {
      if (range == null) return false;
      dispatch(setActive(range.key));
      return true;
    })
    .catch((error) => {
      console.error("Error: ", error);
      return false;
    });
  return false;
};
