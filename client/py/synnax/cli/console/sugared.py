#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, overload, TypeVar
from synnax.cli.console.protocol import Print, Prompt

V = TypeVar("V")

class SugaredPrompt:
    enabled: bool
    prompt: Prompt
    print: Print 

    def __init__(self, prompt: Prompt, print: Print, enabled: bool = True):
        self.prompt = prompt
        self.print = print
        self.enabled = enabled

    @overload
    def ask(
        self,
        question: str,
        choices: list[str] | None = None,
        *,
        arg: str | None = None,
        arg_name: str | None = None,
        default: str = ...,
    ) -> str:
        ...

    @overload
    def ask(
        self,
        question: str,
        choices: list[str] | None = None,
        *,
        arg: str | None = None,
        arg_name: str | None = None,
        default: str | None = None,
    ) -> str | None:
        ...

    @overload
    def ask(
        self,
        question: str,
        choices: list[str] | None = None,
        *,
        arg: str | None = None,
        arg_name: str | None = None,
    ) -> str:
        ... 

    def ask(
        self,
        question: str,
        choices: list[str] | None = None,
        *_,
        **kwargs,
    ) -> str | None:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        v = self.prompt.ask(question, choices, default)
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("This is a required field.")
        return self.ask(question, choices, **kwargs)

    @overload
    def ask_int(
        self,
        question: str,
        bound: tuple[int, int] | None = None,
        *,
        arg: int | None = None,
        arg_name: str | None = None,
        default: int = ...,
    ) -> int:
        ...

    @overload
    def ask_int(
        self,
        question: str,
        bound: tuple[int, int] | None = None,
        *,
        arg: int | None = None,
        arg_name: str | None = None,
        default: int | None = None,
    ) -> int | None:
        ...

    @overload
    def ask_int(
        self,
        question: str,
        bound: tuple[int, int] | None = None,
        *,
        arg: int | None = None,
        arg_name: str | None = None,
    ) -> int:
        ...

    def ask_int(
        self,
        question: str,
        bound: tuple[int, int] | None = None,
        *_,
        **kwargs,
    ) -> int | None:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        v = self.prompt.ask_int(question, bound, default)
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("This is a required field.")
        return self.ask_int(question, **kwargs)

    @overload
    def ask_float(
        self,
        question: str,
        *,
        arg: float | None = None,
        arg_name: str | None = None,
        default: float = ...,
    ) -> float:
        ...

    @overload
    def ask_float(
        self,
        question: str,
        *,
        arg: float | None = None,
        arg_name: str | None = None,
        default: float | None = None,
    ) -> float | None:
        ...

    @overload
    def ask_float(
        self,
        question: str,
        *,
        arg: float | None = None,
        arg_name: str | None = None,
    ) -> float:
        ...

    def ask_float(
        self,
        question: str,
        *_,
        **kwargs,
    ) -> float | None:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        v = self.prompt.ask_float(question, default)
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("This is a required field.")
        return self.ask_float(question, **kwargs)

    @overload
    def confirm(
        self,
        question: str,
        *,
        arg: bool | None = None,
        arg_name: str | None = None,
        default: bool = ...,
    ) -> bool:
        ...

    @overload
    def confirm(
        self,
        question: str,
        *,
        arg: bool | None = None,
        arg_name: str | None = None,
    ) -> bool:
        ...


    def confirm(
        self,
        question: str,
        *_,
        **kwargs,
    ) -> bool:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v # type: ignore
        v = self.prompt.confirm(question, default=default or True)
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("This is a required field.")
        return self.confirm(question, **kwargs)

    
    @overload
    def ask_password(
        self,
        question: str,
        *,
        arg: str | None = None,
        arg_name: str | None = None,
        default: str = ...,
    ) -> str:
        ...

    @overload
    def ask_password(
        self,
        question: str,
        *,
        arg: str | None = None,
        arg_name: str | None = None,
        default: str | None = None,
    ) -> str | None:
        ...

    @overload
    def ask_password(
        self,
        question: str,
        *,
        arg: str | None = None,
        arg_name: str | None = None,
    ) -> str:
        ...

    def ask_password(
        self,
        question: str,
        *_,
        **kwargs,
    ) -> str | None:
        v, default, should_return, has_default = self._validate(kwargs)
        if should_return:
            return v
        v = self.prompt.ask_password(question, default)
        if v is not None or has_default:
            return v
        if self.print is not None:
            self.print.error("This is a required field.")
        return self.ask_password(question, **kwargs)




    def _validate(self, kwargs: dict[str, Any]) -> tuple[V | None, V | None, bool, bool]:
        has_default = "default" in kwargs

        default = kwargs.get("default", None)
        arg = kwargs.get("arg", None)
        if arg is not None:
            return arg, default, True, has_default

        arg_name = kwargs.get("arg_name", None)
        if not self.enabled and not has_default:
            raise ValueError(f"Missing required argument: {arg_name}")

        return default, default, False, has_default

class SugaredConsole(SugaredPrompt):
    print: Print

    def __init__(self, prompt: Prompt, print: Print, enabled: bool = True):
        super().__init__(prompt, print, enabled)

    def info(self, message: str) -> None:
        self.print.info(message)

    def success(self, message: str) -> None:
        self.print.success(message)

    def warn(self, message: str) -> None:
        self.print.warn(message)

    def error(self, message: str) -> None:
        self.print.error(message)

    def table(self, columns: list[str], rows: list[dict]) -> None:
        self.print.table(columns, rows)

        
