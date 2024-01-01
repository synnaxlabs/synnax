import synnax as sy
import matplotlib.pyplot as plt
import numpy as np

client = sy.Synnax()

with client.new_streamer("sy_label_delete") as l:
    for frame in l:
        print(frame["sy_label_delete"][0])
