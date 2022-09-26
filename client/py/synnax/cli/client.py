from .argreader import ArgParser
from .filereader import FileReader

import numpy as np


def run():
    ar = ArgParser()
    filereader = FileReader.get(ar.getFileType(), ar.getFilePath())
    df = filereader.getData()


def initParser(parser):
    parser.add_argument('filepath', type=str,
                        help="filepath of the file to parse")
    parser.add_argument('hostname', type=str, help="IP to synnax node")
    parser.add_argument('port', type=int, default=8080,
                        help="Port of synnax node")
    parser.add_argument('-n', '--testname', type=str, nargs=1,
                        meta_var="testname", help="Name of test")
