// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Access as PAccess, Button, Errors, Icon } from "@synnaxlabs/pluto";
import {
  type ComponentType,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  useCallback,
} from "react";

export interface PermissionsFallbackProps extends Errors.FallbackProps {
  /** Drops the cached permissions and loads them again. */
  retry: () => void;
}

export interface PermissionsBoundaryProps extends PropsWithChildren {
  /** Renders while the permissions load. */
  loading: ReactNode;
  /** Renders when the permissions fail to load. Defaults to the error screen. */
  FallbackComponent?: ComponentType<PermissionsFallbackProps>;
}

// Every guarded surface reads a denial from an empty policy set, so the workspace
// cannot render before the policies land.
const AwaitPermissions = ({ children }: PropsWithChildren): ReactNode => {
  PAccess.useEnsurePermissions({});
  return children;
};

const DefaultFallback = ({
  retry,
  ...rest
}: PermissionsFallbackProps): ReactElement => (
  <Errors.Fallback {...rest}>
    <Button.Button variant="filled" onClick={retry}>
      <Icon.Refresh />
      Retry
    </Button.Button>
  </Errors.Fallback>
);

/** Holds its children back until the permissions of the subject are cached. */
export const PermissionsBoundary = ({
  loading,
  FallbackComponent = DefaultFallback,
  children,
}: PermissionsBoundaryProps): ReactElement => {
  const invalidate = PAccess.useInvalidatePermissions();
  const Fallback = useCallback(
    (props: Errors.FallbackProps): ReactElement => {
      const retry = (): void => {
        invalidate({});
        props.resetErrorBoundary();
      };
      return <FallbackComponent {...props} retry={retry} />;
    },
    [FallbackComponent, invalidate],
  );
  return (
    <Errors.SuspenseBoundary loading={loading} FallbackComponent={Fallback}>
      <AwaitPermissions>{children}</AwaitPermissions>
    </Errors.SuspenseBoundary>
  );
};
