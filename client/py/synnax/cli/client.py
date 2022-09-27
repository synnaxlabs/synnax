import numpy as np

from .argreader import ArgParser
from .filereader import FileReader
from .converter import Converter


def run():
    ar = ArgParser()
    filereader = FileReader.get(ar.getFileType(), ar.getFilePath())
    df = filereader.getData()
    cols = filereader.getHeaders()

    conv = Converter(
        filereader,
        hostname=ar.getHostname(),
        port=ar.getPort(),
        timestampCol=ar.getTimestampCol()[0],
    )
