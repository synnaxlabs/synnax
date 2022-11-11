from typing import Protocol


class Prompt(Protocol):
    """A protocol class for an entity that can prompt the user for input.
    """

    def ask(
        self,
        question: str,
        choices: list[str] = None,
        default: str = None,
    ) -> str:
        """Asks the user a question and returns their response.

        :param question: The question to ask the user.
        :param choices: A list of choices the user can select from. If provided,
        the user will be prompted to select one of the choices. A response not
        in the list will be rejected.
        :param default: A default value to use if the user does not provide a
        response. If provided, the user can press enter to select the default
        value.
        :return: The user's response.
        """
        ...

    def ask_int(
        self,
        question: str,
        default: int = None,
    ) -> int:
        """Asks the user a question and returns their response as an integer.

        :param question: The question to ask the user.
        :param default: A default value to use if the user does not provide a
        """
        ...

    def ask_float(
        self,
        question: str,
        default: float = None,
    ) -> float:
        """Asks the user a question and returns their response as a float.

        :param question: The question to ask the user.
        :param default: A default value to use if the user does not provide a response.
        If provided, the user can press enter to select the default value.
        :return: A float value.
        """
        ...

    def ask_password(
        self,
        question: str,
    ) -> str:
        """Prompts the user for a password and returns their response.
        """
        ...

    def confirm(
        self,
        question: str,
        default: bool = None,
    ) -> bool:
        """Asks the user to confirm the given question.

        :param question: The question to ask the user.
        :param default: A default value to use if the user does not provide a response.
        If provided, the user can press enter to select the default value.
        :return: True if the user confirms the question, False otherwise.
        """
        ...


class Print(Protocol):
    """A protocol class for an entity that can print messages to the console.
    """

    def info(
        self,
        message: str,
    ) -> None:
        """Prints an informational message to the console.

        :param message: The message to print.
        """
        ...

    def warn(
        self,
        message: str,
    ) -> None:
        """Prints a warning message to the console.

        :param message: The message to print.
        """
        ...

    def error(
        self,
        message: str,
    ) -> None:
        """Prints an error message to the console.

        :param message: The message to print.
        """
        ...

    def success(
        self,
        message: str,
    ) -> None:
        """Prints a success message to the console.

        :param message: The message to print.
        """
        ...

    def table(
        self,
        columns: list[str],
        rows: list[dict],
    ):
        """Prints a table to the console.

        :param columns: A list of column names.
        :param rows: A list of dictionaries, where each dictionary represents a row.py. The
        keys of the dictionary should match the column names.
        """
        ...


class Console(Prompt, Print):
    """A protocol class for an entity that can print messages to the console and prompt
    the user for input.
    """
    ...
