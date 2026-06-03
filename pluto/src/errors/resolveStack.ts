// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import StackTrace from "stacktrace-js";

export interface ResolvedStack {
  stack: string;
  componentStack: string | null;
}

/**
 * Resolves the source-map-mapped form of an Error's stack and (optionally) its React
 * component stack into clean, human-readable text.
 *
 * Throws if stacktrace-js cannot resolve either stack (e.g. source maps unavailable,
 * parser cannot recognize the stack format). Callers are expected to handle the
 * rejection and fall back to the raw strings — surfacing the underlying failure is the
 * caller's responsibility, since "resolution failed silently" would mask real
 * deployment problems (missing maps, broken middleware, etc.).
 */
export const resolveStack = async (
  error: Error,
  componentStack: string | null,
): Promise<ResolvedStack> => {
  const [stack, resolvedComponentStack] = await Promise.all([
    resolveFromStack(error),
    componentStack != null ? resolveFromComponentStack(componentStack) : null,
  ]);
  return { stack, componentStack: resolvedComponentStack };
};

const resolveFromStack = async (error: Error): Promise<string> => {
  const frames = await StackTrace.fromError(error);
  return frames.map(formatFrame).join("\n");
};

const resolveFromComponentStack = async (componentStack: string): Promise<string> => {
  const synthetic = new Error();
  synthetic.stack = componentStack;
  const frames = await StackTrace.fromError(synthetic);
  return frames.map(formatFrame).join("\n");
};

const formatFrame = (frame: StackTrace.StackFrame): string => {
  const name = frame.functionName ?? "<anonymous>";
  const location = formatLocation(frame);
  return location != null ? `  at ${name} (${location})` : `  at ${name}`;
};

const formatLocation = (frame: StackTrace.StackFrame): string | null => {
  if (frame.fileName == null) return null;
  const file = cleanFileName(frame.fileName);
  if (frame.lineNumber == null) return file;
  if (frame.columnNumber == null) return `${file}:${frame.lineNumber}`;
  return `${file}:${frame.lineNumber}:${frame.columnNumber}`;
};

// Strip leading "../" segments and "webpack:///" / "webpack-internal:///" prefixes
// that source maps sometimes emit, keeping frames compact.
const cleanFileName = (file: string): string =>
  file.replace(/^webpack(?:-internal)?:\/\/\/?/, "").replace(/^(?:\.\.\/)+/, "");
