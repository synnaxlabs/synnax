import numpy as np

from .argreader import ArgParser
from .filereader import FileReader
from .converter import Converter

import synnax
from datetime import datetime


def run():
    ar = ArgParser()
    filereader = FileReader.get(ar.get_filetype(), ar.get_filepath())

    conv = Converter(
        filereader,
        ar.get_flags(),
        hostname=ar.get_hostname(),
        port=ar.get_port(),
        timestampCol=ar.get_timestamp_col(),
        datarate=25 * synnax.HZ,
    )
    conv.parse()  #
