from typing import Protocol, Any


class Prompt(Protocol):
    def ask(
        self,
        question: str,
        choices: list[str] = None,
        default: str = None,
    ) -> str:
        ...

    def ask_int(
        self,
        question: str,
        default: int = None,
    ) -> int:
        ...

    def ask_float(
        self,
        question: str,
        default: float = None,
    ) -> float:
        ...

    def confirm(
        self,
        question: str,
        default: bool = None,
    ) -> bool:
        ...


class Print(Protocol):
    def info(
        self,
        message: str,
    ) -> None:
        ...

    def warn(
        self,
        message: str,
    ) -> None:
        ...

    def error(
        self,
        message: str,
    ) -> None:
        ...

    def success(
        self,
        message: str,
    ) -> None:
        ...

    def table(
        self,
        columns: list[str],
        rows: list[dict],
    ):
        ...


class Console(Prompt, Print):
    ...
