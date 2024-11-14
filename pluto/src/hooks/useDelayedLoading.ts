// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

/**
 * A custom hook that introduces a delay before setting a `loading` state to `true`.
 * This can help prevent flickering effects in components when the loading state changes
 * quickly.
 *
 * @param loading - The loading state to debounce.
 * @param delay - The delay in milliseconds before setting the loading state to `true`
 * (default is 150ms).
 * @returns `boolean` - Returns `true` if the delayed loading state is active, otherwise
 * `false`.
 */
export const useDelayedLoading = (loading: boolean, delay = 150) => {
  const [isLoading, setIsLoading] = useState(false);
  let loadingTimeout: NodeJS.Timeout;
  useEffect(() => {
    if (loading)
      // Start a delay to set isLoading to true
      loadingTimeout = setTimeout(() => setIsLoading(true), delay);
    else {
      // Clear the timeout if loading becomes false before delay completes
      clearTimeout(loadingTimeout);
      setIsLoading(false);
    }
    // Cleanup on unmount or when loading changes
    return () => clearTimeout(loadingTimeout);
  }, [loading, delay]);
  return isLoading;
};
