import numpy as np

from .argreader import ArgParser
from .filereader import FileReader


def run():
    ar = ArgParser()
    filereader = FileReader.get(ar.getFileType(), ar.getFilePath())
    df = filereader.getData()
