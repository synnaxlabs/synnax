#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import glob
import importlib.util
import inspect
import itertools
import json
import os
import sys
import threading
from collections.abc import Callable
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from framework.target_filter import TargetFilter
from framework.test_case import TestCase


@dataclass
class TestDefinition:
    """A test case definition from a sequence configuration file."""

    case: str
    name: str | None = None
    parameters: dict[str, Any | list[Any]] = field(default_factory=dict)
    matrix: dict[str, list[Any]] | None = None

    @property
    def display_name(self) -> str:
        """The human-readable name: explicit name or last segment of case path."""
        return self.name or self.case.split("/")[-1]

    def __str__(self) -> str:
        if self.name and self.name != self.case.split("/")[-1]:
            return f"{self.case} ({self.name})"
        return self.case


@dataclass
class Sequence:
    """A named group of test definitions with an execution order."""

    name: str
    order: str  # "sequential" | "random" | "asynchronous"
    pool_size: int
    tests: list[TestDefinition]


class ConfigClient:
    """Discovers, loads, filters, and expands test configurations."""

    def __init__(
        self,
        tests_dir: Path | None = None,
        log: Callable[[str], None] | None = None,
    ) -> None:
        if tests_dir is None:
            framework_dir = Path(__file__).resolve().parent
            tests_dir = framework_dir.parent / "tests"
        self._tests_dir = tests_dir
        self._log = log or (lambda _msg: None)
        self._import_lock = threading.Lock()

    def load(
        self,
        target_filter: TargetFilter,
    ) -> tuple[list[Sequence], list[TestDefinition]]:
        """Load, filter, expand, and return sequences and a flat test list.

        Args:
            target_filter: Filter controlling which files, sequences, and
                cases to include.

        Returns:
            A tuple of (sequences, all_test_definitions).
        """
        raw_sequences = self._load_json(target_filter.file_filter)
        sequences = self._process_sequences(raw_sequences, target_filter)

        all_defs: list[TestDefinition] = []
        for seq in sequences:
            all_defs.extend(seq.tests)

        self._validate_unique_names(all_defs)
        self._log(f"Total: {len(all_defs)} tests across {len(sequences)} sequences")
        return sequences, all_defs

    def _validate_unique_names(self, defs: list[TestDefinition]) -> None:
        """Ensure no two test definitions produce the same string key."""
        seen: dict[str, TestDefinition] = {}
        for td in defs:
            key = str(td)
            if key in seen:
                raise ValueError(
                    f"Duplicate test name '{key}' — "
                    f"first in case '{seen[key].case}', "
                    f"duplicate in case '{td.case}'. "
                    f"Use the 'name' field to disambiguate."
                )
            seen[key] = td

    # ----- JSON loading -----

    def _load_json(self, file_filter: list[str] | None) -> list[dict[str, Any]]:
        """Load raw sequence dicts from JSON files."""
        if not file_filter:
            test_files = glob.glob(str(self._tests_dir / "*_tests.json"))
            if not test_files:
                raise FileNotFoundError(
                    "No *_tests.json files found for auto-discovery"
                )
        else:
            test_files = [f"{f}_tests.json" for f in file_filter]

        all_sequences: list[dict[str, Any]] = []
        for test_file in test_files:
            file_path = Path(test_file)
            if not file_path.exists():
                file_path = self._tests_dir / test_file
            if not file_path.exists():
                file_path = Path("tests") / test_file

            try:
                with open(file_path) as f:
                    file_data = json.load(f)
                if "sequences" in file_data:
                    all_sequences.extend(file_data["sequences"])
            except FileNotFoundError:
                raise FileNotFoundError(f"Test file not found: {test_file}")
            except json.JSONDecodeError as e:
                raise ValueError(f"Invalid JSON in test file '{test_file}': {e}") from e

        if not all_sequences:
            raise FileNotFoundError("No valid sequences found")

        return all_sequences

    # ----- Sequence processing -----

    def _process_sequences(
        self,
        sequences_array: list[dict[str, Any]],
        target_filter: TargetFilter,
    ) -> list[Sequence]:
        sequences: list[Sequence] = []

        for seq_idx, seq_dict in enumerate(sequences_array):
            if not isinstance(seq_dict, dict):
                continue

            seq_name = seq_dict.get("sequence_name", f"Sequence_{seq_idx + 1}")
            seq_order: str = seq_dict.get("sequence_order", "sequential").lower()
            pool_size: int = seq_dict.get("pool_size", -1)
            raw_tests: list[dict[str, Any]] = seq_dict.get("tests", [])

            seq_name_matches = target_filter.matches_sequence(seq_name)

            expanded_tests: list[TestDefinition] = []
            for test in raw_tests:
                case_path: str = test["case"]
                if not seq_name_matches and not target_filter.matches_case(case_path):
                    continue

                test_def = TestDefinition(
                    case=case_path,
                    name=test.get("name", None),
                    parameters=test.get("parameters", {}),
                    matrix=test.get("matrix", None),
                )

                class_defs = self._expand_test_classes(test_def)
                for class_def in class_defs:
                    # -f matches against both the file path (e.g. "migration/channels")
                    # and the class name (e.g. "ChannelsSetup"), so you can filter into
                    # individual classes: `tc migration -f setup` matches *Setup classes.
                    expanded = self._expand_parameters(class_def)
                    if target_filter.case_filter is not None:
                        matches_case = target_filter.matches_case(case_path)
                        matches_class = (
                            class_def.name is not None
                            and target_filter.matches_case(class_def.name)
                        )
                        if not matches_case and not matches_class:
                            expanded = [
                                t
                                for t in expanded
                                if t.name is not None
                                and target_filter.matches_case(t.name)
                            ]
                    expanded_tests.extend(expanded)

            if target_filter.exclude is not None:
                expanded_tests = [
                    t
                    for t in expanded_tests
                    if not (
                        target_filter.excluded(t.case)
                        or (t.name is not None and target_filter.excluded(t.name))
                    )
                ]

            if not seq_name_matches and not expanded_tests:
                continue

            if expanded_tests:
                seq = Sequence(
                    name=seq_name,
                    order=seq_order,
                    pool_size=pool_size,
                    tests=expanded_tests,
                )
                sequences.append(seq)

                original_count = len(raw_tests)
                num_expanded = len(expanded_tests)
                if target_filter.case_filter:
                    filter_desc = ",".join(target_filter.case_filter)
                    self._log(
                        f"Loaded sequence '{seq_name}' with {num_expanded} tests "
                        f"matching '{filter_desc}' ({seq_order})"
                    )
                elif num_expanded > original_count:
                    self._log(
                        f"Loaded sequence '{seq_name}' with {original_count} test "
                        f"definitions, expanded to {num_expanded} tests ({seq_order})"
                    )
                else:
                    self._log(
                        f"Loaded sequence '{seq_name}' with {original_count} "
                        f"tests ({seq_order})"
                    )

        if not sequences:
            parts: list[str] = []
            if target_filter.sequence_filter:
                parts.append(f"sequence='{target_filter.sequence_filter}'")
            if target_filter.case_filter:
                parts.append(f"case='{','.join(target_filter.case_filter)}'")
            raise ValueError(f"No tests found matching filters: {', '.join(parts)}")

        return sequences

    # ----- Parameter expansion -----

    def _expand_parameters(self, test_def: TestDefinition) -> list[TestDefinition]:
        if not test_def.parameters:
            return [test_def]

        single_params: dict[str, Any] = {}
        multi_params: dict[str, list[Any]] = {}

        for key, value in test_def.parameters.items():
            if isinstance(value, list):
                multi_params[key] = value
            else:
                single_params[key] = value

        if not multi_params:
            return [test_def]

        param_keys = list(multi_params.keys())
        param_values = [multi_params[key] for key in param_keys]
        combinations = list(itertools.product(*param_values))

        expanded: list[TestDefinition] = []
        for combo in combinations:
            combo_params = dict(zip(param_keys, combo))
            merged_params = {**single_params, **combo_params}

            matrix_suffix = "_".join(str(v) for v in combo)
            base_name = test_def.name or test_def.case.split("/")[-1]
            generated_name = f"{base_name}_{matrix_suffix}"

            expanded.append(
                TestDefinition(
                    case=test_def.case,
                    name=generated_name,
                    parameters=merged_params,
                )
            )

        return expanded

    # ----- Test class expansion -----

    def _expand_test_classes(self, test_def: TestDefinition) -> list[TestDefinition]:
        try:
            test_classes = self._load_test_classes(test_def)

            if len(test_classes) == 1:
                return [test_def]

            expanded_defs: list[TestDefinition] = []
            for test_class in test_classes:
                expanded_defs.append(
                    TestDefinition(
                        case=test_def.case,
                        name=test_class.__name__,
                        parameters=test_def.parameters.copy(),
                        matrix=test_def.matrix,
                    )
                )
            return expanded_defs

        except Exception as e:
            raise ImportError(
                f"Failed to expand test classes for {test_def.case}: {e}"
            ) from e

    # ----- Dynamic class loading -----

    def _load_test_classes(self, test_def: TestDefinition) -> list[type[TestCase]]:
        """Dynamically load TestCase subclass(es) from a case identifier."""
        try:
            case_path = f"tests/{test_def.case}"
            module_name = case_path.split("/")[-1]

            current_dir = os.getcwd()
            script_dir = str(Path(__file__).resolve().parent)

            possible_paths = [
                os.path.join(script_dir, "..", f"{case_path}.py"),
                os.path.join(current_dir, f"{case_path}.py"),
            ]

            file_path: str | None = None
            for path in possible_paths:
                if os.path.exists(path):
                    file_path = path
                    break

            if file_path is None:
                debug_info = (
                    f"\n  cwd: {current_dir}"
                    f"\n  script_dir: {script_dir}"
                    f"\n  case: {test_def.case}"
                    f"\n  tried: {possible_paths}"
                )
                raise FileNotFoundError(
                    f"Could not find test module for {test_def.case}.{debug_info}"
                )

            integration_dir = os.path.dirname(script_dir)

            with self._import_lock:
                if integration_dir not in sys.path:
                    sys.path.insert(0, integration_dir)

                tests_dir = os.path.join(integration_dir, "tests")
                if os.path.isdir(tests_dir):
                    import types

                    tests_pkg = sys.modules.get("tests")
                    if tests_pkg is None or not hasattr(tests_pkg, "__path__"):
                        tests_pkg = types.ModuleType("tests")
                        tests_pkg.__path__ = [tests_dir]
                        tests_pkg.__package__ = "tests"
                        sys.modules["tests"] = tests_pkg
                    elif tests_dir not in tests_pkg.__path__:
                        tests_pkg.__path__.insert(0, tests_dir)

                spec = importlib.util.spec_from_file_location(module_name, file_path)
                if spec is None:
                    raise ImportError(
                        f"Cannot create spec for module: {module_name} at {file_path}"
                    )
                module = importlib.util.module_from_spec(spec)
                if spec.loader is not None:
                    spec.loader.exec_module(module)

            def is_valid_test_case(obj: Any) -> bool:
                try:
                    return (
                        isinstance(obj, type)
                        and not obj.__name__.startswith("_")
                        and issubclass(obj, TestCase)
                        and obj is not TestCase
                        and obj.__module__ == module.__name__
                        # skip classes with unimplemented @abstractmethod
                        and not inspect.isabstract(obj)
                    )
                except (AttributeError, TypeError):
                    return False

            test_classes: list[type[TestCase]] = [
                getattr(module, name)
                for name in dir(module)
                if is_valid_test_case(getattr(module, name))
            ]

            if not test_classes:
                raise AttributeError(f"No TestCase subclass found in {file_path}")

            if test_def.name:
                matching = [
                    cls for cls in test_classes if cls.__name__ == test_def.name
                ]
                if matching:
                    return [matching[0]]

            return test_classes

        except Exception as e:
            raise ImportError(
                f"Failed to load test class(es) from {test_def.case}: {e}\n"
            )

    def load_test_class(self, test_def: TestDefinition) -> type[TestCase]:
        """Load a single test class (first match) from a case identifier."""
        classes = self._load_test_classes(test_def)
        return classes[0]
