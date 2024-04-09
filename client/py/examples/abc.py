
class ABC:
    def __init__(self, value):
        self.value = value


one = ABC(1)
two = ABC(2)
three = ABC(3)

s = {one, two, three}

s.remove(one)

print([x.value for x in s])
