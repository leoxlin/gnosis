import unittest

import calculator


class CalculatorTest(unittest.TestCase):
    def test_subtracts(self):
        self.assertEqual(calculator.subtract(7, 2), 5)


if __name__ == "__main__":
    unittest.main()
