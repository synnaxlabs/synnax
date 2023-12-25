import synnax as sy
import matplotlib.pyplot as plt
from test_complete import test_complete


@test_complete()
def process(s: sy.Range):
    plt.plot(sy.elapsed_seconds(s.gse_ai_time), s.gse_ai_0)
    plt.savefig("test.png")
