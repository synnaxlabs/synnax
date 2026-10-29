// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, errors, map, observe, type record, shallow, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { type Status } from "@/status";

export interface FieldState<V = unknown> {
  value: V;
  status: Status.Crude;
  touched: boolean;
  /** Whether the schema refuses the field emptied: "" for text, [] for a list. */
  required: boolean;
}

// The value a cleared input leaves for a field of this value's type, or undefined when
// no input can clear it.
const emptyValue = (value: unknown): unknown =>
  typeof value === "string" ? "" : Array.isArray(value) ? [] : undefined;

// An asynchronous check cannot run while a field renders, so a schema with one never
// marks its field.
const refusesEmpty = (schema: z.ZodType, value: unknown): boolean => {
  const empty = emptyValue(value);
  if (empty === undefined) return false;
  try {
    return !schema.safeParse(empty).success;
  } catch (e) {
    if (e instanceof z.core.$ZodAsyncError) return false;
    throw errors.fromUnknown(e);
  }
};

export interface RequiredGetOptions {
  optional?: false;
}

export interface GetOptions {
  /** Whether a path the form cannot serve returns null instead of throwing. */
  optional?: boolean;
}

const getVariant = (issue: z.core.$ZodIssue): Status.Variant => {
  if (issue.code === "custom" && issue.params != null && "variant" in issue.params)
    return issue.params.variant;
  return "error";
};

interface SetValueOptions {
  markTouched?: boolean;
}

/**
 * Copies the root and every container on the way to the path's leaf, walking the path
 * the way `deep.set` and `deep.remove` do. A write at the path then gives each ancestor
 * a new identity and leaves every untouched subtree shared.
 */
const copyPath = <V>(root: V, path: string): V => {
  const next = shallow.copy(root);
  const parts = path.split(deep.SEPARATOR);
  let parent = next as record.Unknown;
  let i = 0;
  while (i < parts.length - 1) {
    const best = deep.findBestKey(parent, parts.slice(i, parts.length - 1));
    if (best == null) break;
    const [key, consumed] = best;
    i += consumed;
    const child = deep.defaultGetter(parent, key);
    const copy = shallow.copy(child) as record.Unknown;
    // An array entry found by its key field sits at an index, not at the key.
    if (parent[key] === child) parent[key] = copy;
    else parent[(parent as unknown as unknown[]).indexOf(child)] = copy;
    parent = copy;
  }
  return next;
};

export class State<Z extends z.ZodType> extends observe.Observer<void> {
  private readonly schema?: Z;
  /** The current values. A write copies each object on its path, never changing one. */
  values: z.infer<Z>;
  initialValues: z.infer<Z>;

  private readonly statuses: Map<string, Status.Crude>;
  /** Paths whose status came from `setStatus`, which validation never removes. */
  private readonly external: Set<string>;
  private readonly touched: Set<string>;
  private readonly cachedRefs: Map<string, { state: FieldState; stale: boolean }>;

  constructor(values: z.infer<Z>, schema?: Z) {
    super();
    this.schema = schema;
    this.values = deep.copy(values);
    this.statuses = new Map();
    this.external = new Set();
    this.touched = new Set();
    this.cachedRefs = new Map();
    this.initialValues = deep.copy(this.values);
  }

  setValue(path: string, value: unknown, options?: SetValueOptions) {
    const prev = this.values;
    if (path == "") this.values = deep.copy(value) as z.infer<Z>;
    else {
      this.materialize(path);
      this.write(path, value);
    }
    this.clearChangedExternal(prev);
    this.checkTouched(path, value, options);
    this.updateCachedRefs(path);
  }

  /** Clears each external status whose value differs from its value in prev. */
  private clearChangedExternal(prev: z.infer<Z>) {
    this.external.forEach((path) => {
      const before = deep.get(prev, path, { optional: true });
      const after = deep.get(this.values, path, { optional: true });
      if (before === after || deep.equal(before, after)) return;
      this.deleteStatus(path);
    });
  }

  private write(path: string, value: unknown) {
    this.values = copyPath(this.values, path);
    deep.set(this.values, path, value);
  }

  /** @returns the value the schema supplies for an absent path, or undefined. */
  private schemaDefault(path: string): unknown {
    if (this.schema == null) return undefined;
    const schema = zod.getFieldSchema(this.schema, path, {
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

  /** @returns the first absent ancestor of the path together with its default. */
  private absentAncestor(path: string): [prefix: string, value: unknown] | null {
    const parts = path.split(".");
    for (let i = 1; i < parts.length; i++) {
      const prefix = parts.slice(0, i).join(".");
      if (deep.get(this.values, prefix, { optional: true }) != null) continue;
      const value = this.schemaDefault(prefix);
      return value == null ? null : [prefix, value];
    }
    return null;
  }

  /**
   * Builds the absent ancestor of the path from its schema default, so a write to a
   * leaf never leaves a parent the schema would reject.
   */
  private materialize(path: string) {
    const ancestor = this.absentAncestor(path);
    if (ancestor != null) this.write(...ancestor);
  }

  /** Reads an absent path out of the default a write would materialize. */
  private readAbsent(path: string): unknown {
    const ancestor = this.absentAncestor(path);
    if (ancestor == null) return undefined;
    const [prefix, value] = ancestor;
    const rest = path.slice(prefix.length + 1);
    return deep.get(value, rest, { optional: true }) ?? undefined;
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

  /**
   * Sets the status of the path, keyed by the path. Validation never removes it. It
   * clears when the value at the path changes, when validation sets a status there, or
   * through `clearStatus`.
   */
  setStatus(path: string, status: Status.Crude) {
    this.statuses.set(path, { ...status, key: path });
    this.external.add(path);
    this.updateCachedRefs(path);
  }

  /** Clears the status of the path, or of every path when it is empty. */
  clearStatus(path: string = "") {
    if (path !== "") return this.deleteStatus(path);
    this.statuses.clear();
    this.external.clear();
    this.updateCachedRefs(path);
  }

  /** Deletes the status of the path alone, even when the path is the root. */
  private deleteStatus(path: string) {
    this.statuses.delete(path);
    this.external.delete(path);
    this.updateCachedRefs(path);
  }

  reset(initialValues?: z.infer<Z>) {
    if (deep.equal(this.values, initialValues ?? this.initialValues)) return;
    if (initialValues != null) this.initialValues = deep.copy(initialValues);
    const nextValues = deep.copy(this.initialValues);
    this.statuses.clear();
    this.external.clear();
    this.touched.clear();
    this.cachedRefs.clear();
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
    const prev = this.values;
    this.values = copyPath(this.values, path);
    deep.remove(this.values, path);
    this.clearChangedExternal(prev);
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
    accumulated: Status.Crude[],
  ): Status.Crude[] {
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
    const next = new Map<string, Status.Crude>();
    let success = true;
    if (!result.success)
      this.errorsToStatuses(result.error.issues, [], []).forEach((status) => {
        const { key: issuePath, message, variant } = status;
        if (issuePath == null) return;
        const matchesPath =
          validationPath == null ||
          deep.pathsMatch(issuePath, validationPath) ||
          deep.pathsMatch(validationPath, issuePath);
        const isTouched = validateUntouched || this.touched.has(issuePath);
        if (!matchesPath || !isTouched) return;
        if (variant !== "warning") success = false;
        next.set(issuePath, { key: issuePath, variant, message });
      });
    this.statuses.forEach((status, path) => {
      if (status.variant === "success" || this.external.has(path) || next.has(path))
        return;
      this.deleteStatus(path);
    });
    // A status that did not change keeps its object, so its field does not re-render.
    next.forEach((status, path) => {
      this.external.delete(path);
      const prev = this.statuses.get(path);
      if (prev?.variant === status.variant && prev.message === status.message) return;
      this.statuses.set(path, status);
      this.updateCachedRefs(path);
    });
    return success;
  }

  get hasBeenTouched() {
    return this.touched.size > 0;
  }

  getStatuses(): Status.Crude[] {
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
    const cached = this.cachedRefs.get(path);
    if (cached != null && !cached.stale) return cached.state as FieldState<V>;
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
      if (value == null) {
        const verdict = fieldSchema.safeParse(undefined);
        if (!verdict.success) {
          if (optional) return null;
          throw new Error(`Field ${path} is required and holds no value`);
        }
        value = verdict.data as V;
      }
      required = refusesEmpty(fieldSchema, value);
    } else if (value == null) {
      if (optional) return null;
      throw new Error(`Field ${path} is not in the form`);
    }
    const status = map.getOrSetDefault(this.statuses, path, {
      key: path,
      variant: "success",
      message: "",
    });
    const touched = this.touched.has(path);
    // A field whose state did not change keeps its object, so it does not re-render.
    if (
      cached != null &&
      cached.state.value === value &&
      cached.state.required === required &&
      cached.state.status === status &&
      cached.state.touched === touched
    ) {
      cached.stale = false;
      return cached.state as FieldState<V>;
    }
    const state: FieldState<V> = { value, required, status, touched };
    this.cachedRefs.set(path, { state, stale: false });
    return state;
  }

  /**
   * Marks stale the cached state of the path, its ancestors, and everything under its
   * parent. A sibling can be the discriminator that decides the path's schema.
   */
  private updateCachedRefs(fieldPath: string) {
    const parent = fieldPath.slice(0, Math.max(fieldPath.lastIndexOf("."), 0));
    this.cachedRefs.forEach((cached, refPath) => {
      if (deep.pathsMatch(refPath, parent) || deep.pathsMatch(fieldPath, refPath))
        cached.stale = true;
    });
  }
}
