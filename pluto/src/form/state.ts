// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { deep, map, observe, zod } from "@synnaxlabs/x";
import { type z } from "zod";

export interface FieldState<V = unknown> {
  value: V;
  status: status.Crude;
  touched: boolean;
  required: boolean;
}

export interface RequiredGetOptions {
  optional?: false;
}

export interface GetOptions {
  /** Whether a path the form cannot serve returns null instead of throwing. */
  optional?: boolean;
}

const getVariant = (issue: z.core.$ZodIssue): status.Variant => {
  if (issue.code === "custom" && issue.params != null && "variant" in issue.params)
    return issue.params.variant;
  return "error";
};

interface SetValueOptions {
  markTouched?: boolean;
}

export class State<Z extends z.ZodType> extends observe.Observer<void> {
  private readonly schema?: Z;
  values: z.infer<Z>;
  initialValues: z.infer<Z>;

  private readonly statuses: Map<string, status.Crude>;
  private readonly touched: Set<string>;
  private readonly cachedRefs: Map<string, {}>;

  constructor(values: z.infer<Z>, schema?: Z) {
    super();
    this.schema = schema;
    this.values = deep.copy(values);
    this.statuses = new Map();
    this.touched = new Set();
    this.cachedRefs = new Map();
    this.initialValues = deep.copy(this.values);
  }

  setValue(path: string, value: unknown, options?: SetValueOptions) {
    if (path == "") this.values = deep.copy(value) as z.infer<Z>;
    else {
      this.materialize(path);
      deep.set(this.values, path, value);
    }
    this.checkTouched(path, value, options);
    this.updateCachedRefs(path);
  }

  /** @returns the value the schema supplies for an absent prefix, or undefined. */
  private schemaDefault(prefix: string): unknown {
    if (this.schema == null) return undefined;
    const schema = zod.getFieldSchema(this.schema, prefix, {
      optional: true,
      values: this.values,
    });
    if (schema == null) return undefined;
    const parsed = schema.safeParse(undefined);
    if (parsed.success && parsed.data != null) return parsed.data;
    // An optional object counts as defaulted when its own fields can complete it.
    const inner = zod.unwrap(schema);
    if (inner._zod.def.type !== "object") return undefined;
    const built = (inner as z.ZodObject).safeParse({});
    return built.success ? built.data : undefined;
  }

  /**
   * Builds each absent ancestor of the path from its schema default, so a write to a
   * leaf never leaves a parent the schema would reject.
   */
  private materialize(path: string) {
    const parts = path.split(".");
    for (let i = 1; i < parts.length; i++) {
      const prefix = parts.slice(0, i).join(".");
      if (deep.get(this.values, prefix, { optional: true }) != null) continue;
      const value = this.schemaDefault(prefix);
      if (value == null) return;
      deep.set(this.values, prefix, value);
    }
  }

  /**
   * Reads an absent path the way a write would materialize it: the nearest absent
   * ancestor with a schema default supplies the rest of the path.
   */
  private readAbsent(path: string): unknown {
    const parts = path.split(".");
    let base: unknown = this.values;
    let start = 0;
    for (let i = 1; i < parts.length; i++) {
      const rel = parts.slice(start, i).join(".");
      if (deep.get(base, rel, { optional: true }) != null) continue;
      const value = this.schemaDefault(parts.slice(0, i).join("."));
      if (value == null) return undefined;
      [base, start] = [value, i];
    }
    if (start === 0) return undefined;
    return (
      deep.get(base, parts.slice(start).join("."), { optional: true }) ?? undefined
    );
  }

  private checkTouched(path: string, value: unknown, options: SetValueOptions = {}) {
    const { markTouched = true } = options;
    const initialValue = deep.get(this.initialValues, path, { optional: true });
    const equalsInitial = deep.equal(initialValue, value);
    if (equalsInitial) {
      this.clearTouched(path);
      // When a value equals its initial value, we should also clear
      // any touched child paths that no longer exist in the current value
      this.clearOrphanedTouchedPaths(path);
    } else if (markTouched) this.setTouched(path);
  }

  private clearOrphanedTouchedPaths(parentPath: string) {
    const pathsToRemove = new Set<string>();
    this.touched.forEach((touchedPath) => {
      if (
        deep.pathsMatch(touchedPath, parentPath) &&
        touchedPath !== parentPath &&
        !deep.has(this.values, touchedPath)
      )
        pathsToRemove.add(touchedPath);
    });
    pathsToRemove.forEach((path) => {
      this.touched.delete(path);
      this.cachedRefs.delete(path);
    });
  }

  setStatus(path: string, status: status.Crude) {
    this.statuses.set(path, status);
    this.updateCachedRefs(path);
  }

  clearStatus(path: string = "") {
    this.statuses.delete(path);
    this.updateCachedRefs(path);
  }

  reset(initialValues?: z.infer<Z>) {
    if (deep.equal(this.values, initialValues ?? this.initialValues)) return;
    if (initialValues != null) this.initialValues = deep.copy(initialValues);
    const nextValues = deep.copy(this.initialValues);
    this.statuses.clear();
    this.touched.clear();
    const cachedRefsToClear = new Set<string>();
    this.cachedRefs.forEach((_, path) => {
      const prev = deep.get(this.values, path, { optional: true });
      const next = deep.get(nextValues, path, { optional: true });
      if (!deep.equal(prev, next)) cachedRefsToClear.add(path);
    });
    cachedRefsToClear.forEach((path) => this.cachedRefs.delete(path));
    this.values = nextValues;
  }

  setCurrentStateAsInitialValues() {
    this.initialValues = deep.copy(this.values);
    this.touched.clear();
    this.cachedRefs.clear();
  }

  setTouched(path: string) {
    this.touched.add(path);
    this.updateCachedRefs(path);
  }

  clearTouched(path: string = "") {
    if (path === "") this.touched.clear();
    else this.touched.delete(path);
    this.updateCachedRefs(path);
    this.notify();
  }

  remove(path: string) {
    deep.remove(this.values, path);
    this.statuses.delete(path);
    this.clearTouched(path);
    this.cachedRefs.delete(path);
  }

  validate(validateUntouched?: boolean, path?: string): boolean {
    if (this.schema == null) return true;
    const result = this.schema.safeParse(this.values);
    return this.processValidation(validateUntouched, result, path);
  }

  async validateAsync(validateUntouched?: boolean, path?: string): Promise<boolean> {
    if (this.schema == null) return true;
    const result = await this.schema.safeParseAsync(this.values);
    return this.processValidation(validateUntouched, result, path);
  }

  private errorsToStatuses(
    issues: z.core.$ZodIssue[],
    path: PropertyKey[],
    accumulated: status.Crude[],
  ): status.Crude[] {
    issues.forEach((issue) => {
      const issuePath = [...path, ...issue.path];
      const resolvedPath = deep.resolvePath(issuePath.join("."), this.values);
      accumulated.push({
        key: resolvedPath,
        variant: getVariant(issue),
        message: issue.message,
      });
      if (
        issue.code === "invalid_union" &&
        issue.errors != null &&
        issue.errors.length > 0
      ) {
        // If we're operating on a union, we need to find the most favored error by
        // finding the error with the least number of errors whose path exists in the
        // current value, but has at least one error.
        const rankedIssues = issue.errors.map((subIssues) =>
          subIssues.filter((e) =>
            deep.has(this.values, [...issuePath, ...e.path].join(".")),
          ),
        );
        rankedIssues.sort((a, b) => a.length - b.length);
        const candidateError = rankedIssues.find((i) => i.length > 0);
        if (candidateError == null) return;
        this.errorsToStatuses(candidateError, issuePath, accumulated);
      }
    });
    return accumulated;
  }

  private processValidation(
    validateUntouched: boolean = false,
    result: z.ZodSafeParseResult<z.infer<Z>>,
    validationPath?: string,
  ): boolean {
    if (this.schema == null) return true;
    const cachedRefsToClear = new Set<string>();
    this.statuses.forEach((status, childPath) => {
      if (status.variant !== "success") cachedRefsToClear.add(childPath);
    });
    cachedRefsToClear.forEach((path) => this.clearStatus(path));
    if (result.success) return true;
    let success = true;
    const statuses = this.errorsToStatuses(result.error.issues, [], []);
    statuses.forEach((status) => {
      const { key: issuePath, message, variant } = status;
      if (issuePath == null) return;
      const matchesPath =
        validationPath == null ||
        deep.pathsMatch(issuePath, validationPath) ||
        deep.pathsMatch(validationPath, issuePath);
      const isTouched = validateUntouched || this.touched.has(issuePath);
      if (!matchesPath || !isTouched) return;
      if (variant !== "warning") success = false;
      this.setStatus(issuePath, { key: issuePath, variant, message });
    });
    return success;
  }

  get hasBeenTouched() {
    return this.touched.size > 0;
  }

  getStatuses(): status.Crude[] {
    return Array.from(this.statuses.values()).filter(
      (status) => status.variant !== "success",
    );
  }

  getState<V>(path: string, opts?: RequiredGetOptions): FieldState<V>;
  getState<V>(path: string, opts?: GetOptions): FieldState<V> | null;

  /**
   * Reads the state at a path. An absent value takes the schema's verdict: a default
   * is shown without being written, an optional field reads as `undefined`, and a
   * required one throws.
   *
   * @throws {Error} if the path holds no value, the schema does not supply one, and
   * `optional` is not set.
   */
  getState<V>(path: string, opts: GetOptions = {}): FieldState<V> | null {
    const { optional = false } = opts;
    const cachedRef = map.getOrSetDefault(this.cachedRefs, path, {}) as FieldState<V>;
    let value = deep.get<V, z.infer<Z>>(this.values, path, { optional: true });
    value ??= this.readAbsent(path) as V | null;
    let required = false;
    const fieldSchema =
      this.schema == null
        ? null
        : zod.getFieldSchema(this.schema, path, {
            optional: true,
            values: this.values,
          });
    if (fieldSchema != null) {
      const verdict = fieldSchema.safeParse(undefined);
      required = !verdict.success;
      if (value == null) {
        if (!verdict.success) {
          if (optional) return null;
          throw new Error(`Field ${path} is required and holds no value`);
        }
        value = verdict.data as V;
      }
    } else if (value == null) {
      if (optional) return null;
      throw new Error(`Field ${path} is not in the form`);
    }
    cachedRef.value = value;
    cachedRef.required = required;
    cachedRef.status = map.getOrSetDefault(this.statuses, path, {
      key: path,
      variant: "success",
      message: "",
    });
    cachedRef.touched = this.touched.has(path);
    return cachedRef;
  }

  private updateCachedRefs(fieldPath: string) {
    this.cachedRefs.forEach((_, refPath) => {
      if (deep.pathsMatch(refPath, fieldPath) || deep.pathsMatch(fieldPath, refPath))
        this.cachedRefs.set(refPath, {});
    });
  }
}
