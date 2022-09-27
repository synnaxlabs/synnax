from typing import Protocol

from synnax import telem
from synnax.exceptions import (
    ContiguityError,
    UnexpectedError,
    ValidationError,
    Field,
)

from .sugared import NumpySegment


class Validator(Protocol):
    def validate(self, segment: NumpySegment) -> None:
        ...


class ScalarTypeValidator:
    def validate(self, seg: NumpySegment) -> None:
        npt = seg.channel.data_type.numpy_type
        if npt is None:
            raise ValidationError(
                Field(
                    "data",
                    f"Data type {seg.channel.data_type} is not supported by this writer.",
                )
            )

        if seg.data.dtype != npt:
            raise ValidationError(
                Field("data", f"Expected data type {npt}, got {seg.data.dtype}")
            )

        if seg.data.ndim != 1:
            raise ValidationError(
                Field("data", f"Expected 1D array, got {seg.data.ndim}D array")
            )


class ContiguityValidator:
    high_water_marks: dict[str, telem.TimeStamp]
    allow_no_high_water_mark: bool = False
    allow_overlap: bool = False
    allow_gap: bool = False

    def __init__(
        self,
        high_water_marks: dict[str, telem.TimeStamp],
        allow_no_high_water_mark: bool = False,
        allow_overlap: bool = False,
        allow_gap: bool = False,
    ) -> None:
        self.allow_no_high_water_mark = allow_no_high_water_mark
        self.allow_overlap = allow_overlap
        self.allow_gap = allow_gap
        self.high_water_marks = high_water_marks

    def validate(self, seg: NumpySegment) -> None:
        hwm = self._get_high_water_mark(seg.channel.key)
        if hwm is not None:
            self._enforce_no_overlap(hwm, seg)
            self._enforce_no_gap(hwm, seg)
        self._update_high_water_mark(seg)

    def _enforce_no_overlap(self, hwm: telem.TimeStamp, seg: NumpySegment) -> None:
        if self.allow_overlap:
            return
        if seg.start.before(hwm):
            raise ContiguityError(
                f"Next segment start ({seg.start}) is before the previous segments end ({hwm})"
            )

    def _enforce_no_gap(self, hwm: telem.TimeStamp, seg: NumpySegment) -> None:
        if self.allow_gap:
            return
        if seg.start != hwm:
            raise ContiguityError(
                f"Next segment start ({seg.start}) is not equal to the previous segments end ({hwm})"
            )

    def _update_high_water_mark(self, seg: NumpySegment) -> None:
        self.high_water_marks[seg.channel.key] = seg.end

    def _get_high_water_mark(self, channel_key: str) -> telem.TimeStamp:
        hwm = self.high_water_marks.get(channel_key, None)
        if hwm is None and not self.allow_no_high_water_mark:
            raise UnexpectedError(f"No high water mark for channel: {channel_key}")
        return hwm
