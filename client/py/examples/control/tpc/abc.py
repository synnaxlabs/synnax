import numpy as np
import matplotlib.pyplot as plt
from scipy.signal import find_peaks

# Parameters for the sine wave
t = np.linspace(0, 10, 500)  # Time vector
frequency = 1.0  # Frequency of the sine wave
amplitude = 1.0  # Amplitude of the sine wave
phase = 0        # Phase shift
noise_level = 0.2  # Noise level

# Generate the sine wave
signal = amplitude * np.sin(2 * np.pi * frequency * t + phase)

# Add random noise
noise = noise_level * np.random.normal(size=t.size)
noisy_signal = signal + noise

# Find peaks
peaks, _ = find_peaks(noisy_signal, prominence=0.5, height=2)  # Adjust for better peak
# detection

# Plotting the results
plt.figure(figsize=(10, 4))
plt.plot(t, noisy_signal, label="Noisy Sine Wave")
plt.plot(t[peaks], noisy_signal[peaks], "x", label="Peaks")
plt.title("Noisy Sine Wave with Detected Peaks")
plt.xlabel("Time")
plt.ylabel("Amplitude")
plt.legend()
plt.show()
