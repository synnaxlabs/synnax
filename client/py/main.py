import numpy as np

# create a numpy array of 10 float64 values,
# convert them to bytes, and flush them to a file
arr = np.arange(10, dtype=np.float64)
b = arr.tobytes()
with open("test.bin", "wb") as f:
    f.write(b)