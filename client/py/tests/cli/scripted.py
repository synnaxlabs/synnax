#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io

from synnax.cli.prompt import Prompt


class Scripted:
    """Scripted holds a Prompt whose user gives answers in order, and records what the
    Prompt asks and prints. Asking after the last answer raises EOFError.
    """

    prompt: Prompt
    questions: list[str]
    out: io.StringIO
    err: io.StringIO

    def __init__(self, *answers: object):
        self._answers = [str(a) for a in answers]
        self.questions = []
        self.out = io.StringIO()
        self.err = io.StringIO()
        self.prompt = Prompt(
            read=self._read, read_secret=self._read, out=self.out, err=self.err
        )

    def _read(self, question: str) -> str:
        self.questions.append(question)
        if not self._answers:
            raise EOFError
        return self._answers.pop(0)
