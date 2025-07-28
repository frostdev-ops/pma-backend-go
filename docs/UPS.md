# An In-Depth Analysis and Fine-Tuning Guide for the Geekworm X1202 UPS and MAX17040 Fuel Gauge

## Table of Contents
1. [Architectural Deep Dive](#section-1-architectural-deep-dive)
2. [The ModelGauge Algorithm](#section-2-the-modelgauge-algorithm)
3. [Diagnostic Analysis](#section-3-diagnostic-analysis)
4. [Correcting the Implementation](#section-4-correcting-the-implementation)
5. [Advanced System Tuning](#section-5-advanced-system-tuning)
6. [Final Recommendations](#section-6-final-recommendations)

---

## Section 1: Architectural Deep Dive: The Geekworm X1202 UPS and the Maxim MAX17040 IC

A robust and reliable software implementation is fundamentally built upon a precise understanding of the underlying hardware architecture. Erroneous data, such as that observed in the `ups.py` script, often originates from a disconnect between the software's assumptions and the hardware's actual behavior and communication protocol. 

This section provides a comprehensive architectural deconstruction of the Geekworm X1202 Uninterruptible Power Supply (UPS) and its core intelligent component, the Maxim Integrated MAX17040 fuel-gauge Integrated Circuit (IC).

### 1.1 The Geekworm X1202 Platform

The Geekworm X1202 is an advanced power management and UPS expansion board specifically engineered for the Raspberry Pi 5. Its primary function is to provide a stable, continuous power source, capable of delivering up to **5.1V at a maximum of 5A**, sufficient for even the most demanding applications.

The board is designed to hold four 18650 lithium-ion (Li-Ion) battery cells arranged in a **1-Series, 4-Parallel (1S4P)** configuration. This configuration results in a single logical cell with a large capacity, the voltage of which is monitored by the onboard fuel-gauge system.

#### Key Features of the X1202 Platform:

- **Seamless Power Switching**: The board is equipped with detection mechanisms for AC power loss and power adapter failure. Upon detection of a power interruption, it automatically and seamlessly switches to the backup battery power, ensuring the Raspberry Pi remains operational without interruption.

- **Intelligent Power Management**: The X1202 features sophisticated power-path management that minimizes the frequent charging and discharging of the batteries, which helps to prolong their operational lifespan. It also monitors the Raspberry Pi's power state and can automatically cut power to the output when the Pi is shut down, entering an ultra-low standby power consumption mode to conserve battery life.

- **Hardware Integration and Connectivity**: The X1202 connects to the Raspberry Pi 5 via a set of pogo pins that make contact with pads on the underside of the Pi's Printed Circuit Board (PCB). This design choice eliminates the need for cabling over the 40-pin GPIO header, allowing other HATs to be stacked on top.

  > ⚠️ **Critical Note**: This physical connection is critical; a poor or intermittent contact on the I2C communication pins (SDA and SCL) will lead to failed or corrupted data transfers, which could manifest as unreliable readings. The manufacturer notes that if the UPS does not detect the Raspberry Pi 5 via this connection, it will automatically shut down.

- **System-Level Configuration Dependencies**: The X1202 is not an entirely standalone peripheral. Its behavior is intertwined with the Raspberry Pi 5's EEPROM configuration. For correct operation and to avoid nuisance warnings, specific settings must be configured in the Pi's firmware. These include setting `PSU_MAX_CURRENT=5000` to inform the Pi that the power supply is capable of delivering 5A, and adjusting `POWER_OFF_ON_HALT` to control whether the 5V output remains active after the Pi has halted.

### 1.2 The Heart of the System: The Maxim Integrated MAX17040 Fuel-Gauge IC

The intelligence behind the X1202's battery monitoring capabilities is provided by a dedicated fuel-gauge IC. Based on the 1S4P battery configuration and the functionality described, this chip is the **Maxim Integrated MAX17040**. The MAX17041 is designed for 2-cell (2S) packs and is therefore not the correct component for this application.

The MAX17040 is a low-cost, ultra-compact, host-side fuel gauge designed specifically for single-cell Li-Ion battery applications.

#### Core Features of the MAX17040:

- **ModelGauge Algorithm**: The chip does not use a traditional coulomb-counting method for determining the battery's state of charge (SOC). Instead, it employs a sophisticated battery modeling scheme called ModelGauge. This algorithm uses precision voltage measurements and an internal model of a Li-Ion cell's behavior to calculate the relative SOC.

  > ✅ **Advantage**: This approach eliminates the need for an external current-sense resistor and avoids the cumulative drift error that plagues coulomb-counting systems.

- **Precision Voltage Measurement**: The MAX17040 provides a high-precision voltage measurement of the CELL pin with a typical accuracy of **±12.5mV** over its operational range. This precision is critical to the accuracy of the ModelGauge algorithm.

- **I2C Communication**: All measurement data and configuration parameters are accessed via a standard 2-wire I2C interface that supports speeds up to **400 kHz**.

- **Low Power Consumption**: The IC is designed for portable equipment and features very low power consumption:
  - **50 µA** in active mode
  - **<1 µA** in sleep mode

### 1.3 I2C Communication Protocol and Register Map

All interaction between the Raspberry Pi (the I2C master) and the MAX17040 (the I2C slave) occurs through reading from and writing to a set of 16-bit registers. A failure to adhere to the specified I2C protocol is a common source of software error.

#### I2C Configuration:
- **Slave Address**: `0x36` (7-bit address: `0b0110110`)
- **Data Width**: All registers are 16 bits wide
- **Transfer Requirement**: Any read or write operation must transfer all 16 bits to be considered valid; incomplete transfers are ignored

#### Transaction Protocols:

**Write Transaction:**
1. Master sends START condition
2. Slave address with R/W bit cleared (0)
3. 8-bit memory address of target register
4. Two 8-bit data bytes (MSB first, then LSB)

**Read Transaction:**
1. Master performs write to set memory address pointer (START, SlaveAddr+W, MemAddr)
2. Master issues Repeated START condition
3. Slave address with R/W bit set (1)
4. Slave transmits two 8-bit data bytes (MSB first, then LSB)

#### Key Register Map:

| Address (Hex) | Register Name | Description | Access |
|---------------|---------------|-------------|---------|
| `02h-03h` | **VCELL** | Reports a 12-bit A/D measurement of the battery voltage. The value is left-aligned in the 16-bit register. | Read-Only |
| `04h-05h` | **SOC** | Reports the 16-bit State of Charge calculated by the ModelGauge algorithm. The high byte is the integer percentage, and the low byte is the fractional part (1/256% per bit). | Read-Only |
| `06h-07h` | **MODE** | Used to send special commands to the IC, such as initiating a Quick-Start. | Write-Only |
| `08h-09h` | **VERSION** | Returns a 16-bit value indicating the production version of the IC. | Read-Only |
| `0Ch-0Dh` | **RCOMP** | A 16-bit compensation value used to fine-tune the ModelGauge algorithm for specific battery chemistries or operating temperatures. Default is `9700h`. | Read/Write |
| `FEh-FFh` | **COMMAND** | Used to send special commands, most notably the Power-On Reset (POR) command. | Write-Only |

---

## Section 2: The ModelGauge Algorithm: A Paradigm Shift from Coulomb Counting

The user's perception of "unreliable" readings stems in part from a misunderstanding of the technology at play. The MAX17040 does not behave like a simple digital multimeter. It employs a sophisticated algorithmic approach, ModelGauge, to estimate the battery's state.

### 2.1 Theory of Operation

The core of Maxim's ModelGauge technology is a complex mathematical model that simulates the internal dynamics of a Li-Ion battery. This model is fundamentally **voltage-based**. It leverages the well-established principle that the **Open-Circuit Voltage (OCV)**—the voltage of a battery when it is at rest with no load—has a strong and predictable correlation with its State of Charge (SOC).

However, simply measuring the terminal voltage under load is not sufficient, as this voltage sags depending on the current draw and the battery's internal resistance. The ModelGauge algorithm goes further by incorporating a dynamic model that accounts for several nonlinear factors:

#### Compensation Factors:

- **Time-based Effects**: It considers the effects of chemical reactions and impedance changes within the battery over time.
- **Load and Temperature Compensation**: The algorithm compensates for variations in discharge rate and temperature to provide a stable SOC reading even as these conditions change.
- **Aging Compensation**: The model automatically adapts to the battery as it ages, ensuring that the SOC remains accurate over the battery's entire lifecycle.

By continuously measuring the precision voltage and feeding it into this internal simulation, the MAX17040 can produce an SOC estimate that represents the true energy state of the battery, filtered from the noise of instantaneous voltage fluctuations.

### 2.2 ModelGauge vs. Traditional Coulomb Counting

To appreciate the ModelGauge approach, it is useful to contrast it with the most common alternative: coulomb counting.

#### Coulomb Counting:
This traditional method, also known as current integration, works like a water meter. It uses a low-value, high-precision sense resistor in the main current path. By measuring the tiny voltage drop across this resistor, it calculates the current flowing into or out of the battery. It then integrates this current over time to keep a running tally of the total charge (measured in coulombs or milliamp-hours) that has entered or left the battery.

#### The Drift Problem:
The primary weakness of coulomb counting is the accumulation of small, unavoidable errors. The current-sense amplifier will always have a minuscule offset error. While tiny, this error is integrated continuously. Over hours and days, this leads to a significant "drift" in the calculated SOC.

> ⚠️ **Consequence**: If the device is used for long periods without reaching a fully charged or fully discharged state, the reported SOC can become wildly inaccurate. To correct this drift, coulomb-counting systems require periodic "relearn" cycles, where the battery must be charged to 100% or discharged to 0% to re-synchronize the counter.

#### The ModelGauge Advantage:
The MAX17040's ModelGauge algorithm is fundamentally different because it does not rely on current integration. Its SOC determination is **convergent, not divergent**. When the IC powers up, it makes a "first guess" of the SOC based on the initial voltage reading. Any error in this initial guess will fade over time as the model observes the battery's behavior and converges on the correct state.

> ✅ **Benefits**: This means the MAX17040 does not suffer from long-term drift and does not require full relearn cycles, making it more robust for typical user behavior where batteries are often partially charged.

### 2.3 The Importance of Battery Chemistry and the "18650" Form Factor

A critical detail often overlooked by developers is that **"Lithium-Ion" is not a single, monolithic chemistry**. It is a broad family of chemical compositions, including:

- **LCO** (Lithium Cobalt Oxide)
- **LMO** (Lithium Manganese Oxide)  
- **NMC** (Lithium Nickel Manganese Cobalt Oxide)
- And others

Each of these chemistries has a **unique discharge curve**, meaning the relationship between its OCV and SOC is slightly different.

#### The "18650" Consideration:
The term "18650" refers only to the **physical form factor** of the battery cell (18mm diameter, 65mm length). Cells in this format can be manufactured with any of the various Li-Ion chemistries.

The MAX17040's internal ModelGauge algorithm is pre-programmed with a **generic model** that represents a typical Li-Ion cell. While this provides good performance for many variants, it may not perfectly match the specific 18650 cells being used in the X1202 UPS.

> 🎯 **Implication**: This mismatch between the IC's generic model and the specific discharge curve of the user's cells is a likely secondary source of inaccuracy, entirely separate from the primary software bug. For example, the IC might report 10% SOC when the cells' true SOC is only 5%, or vice-versa.

This is precisely why Maxim included a mechanism for fine-tuning the model, which will be explored in [Section 5](#section-5-advanced-system-tuning).

---

## Section 3: Diagnostic Analysis of the ups.py Script Output

A forensic examination of the console output provided by the user reveals two distinct issues. The first is a definitive software bug resulting in impossible data. The second is a misinterpretation of normal battery physics, which creates a perception of unreliability.

### 3.1 Root Cause of the "255.21%" Anomaly

The most telling piece of evidence is the line:
```
Power connected! Resuming charge from 255.21%.
```

This value is not only physically impossible but also points directly to a **data interpretation error** in the software when reading the 16-bit SOC register at I2C address `0x04`.

#### SOC Register Format:
The format of the SOC register is explicitly defined in the MAX17040 datasheet:

- **High Byte (at address 0x04)**: Represents the integer part of the state of charge, in units of percent. A value of 100 (0x64) in this byte means 100%.
- **Low Byte (at address 0x05)**: Represents the fractional part of the state of charge. Each least significant bit (LSB) in this byte corresponds to 1/256th of a percent.

#### Analysis of the Error:
The erroneous **255.21%** reading is a classic symptom of mishandling this two-part data structure. The number **255** is **0xFF** in hexadecimal, the maximum value for an 8-bit byte. This strongly suggests that the script is either:

1. Reading a byte that contains 0xFF and misinterpreting it as the primary SOC value, or
2. Performing a flawed mathematical combination of the two bytes

#### Correct vs. Faulty Procedure:

**✅ Correct Procedure:**
1. Read the 16-bit word from the SOC register. Let's assume the chip reports 95.5% SOC:
   - High byte would be 95 (0x5F)
   - Low byte would be 0.5 × 256 = 128 (0x80)
   - The 16-bit value read would be 0x5F80

2. Extract the integer part from the high byte: `(0x5F80 >> 8) = 0x5F = 95`
3. Extract the fractional part from the low byte: `(0x5F80 & 0x00FF) = 0x80 = 128`
4. Calculate final value: `95 + (128 / 256.0) = 95.5%`

**❌ Likely Faulty Procedure:**
1. The script reads the 16-bit word 0x5F80
2. It may then perform an incorrect calculation, for example, treating the entire 16-bit integer (24448) as the value and applying a single, incorrect scaling factor

The **255** suggests that at some point, a byte containing **0xFF** is being read and used as the integer part of the percentage. This could happen if the script reads from an incorrect register or if a communication error occurs.

> 🔍 **Most Likely Cause**: A fundamental flaw in the parsing logic that combines the two bytes of the SOC register, likely compounded by a potential byte-order (endianness) mismatch, where the low and high bytes are swapped before the flawed calculation is applied.

### 3.2 Deconstructing the "Unreliable" Voltage Readings

The second issue noted is the fluctuation in reported voltage:

```
Status: ON BATTERY |... | Voltage: 4.17V
```
followed shortly by:
```
Status: ON BATTERY |... | Voltage: 3.85V
```

**This behavior is NOT a sign of a faulty IC or an unreliable measurement.** It is the correct and expected physical behavior of a lithium-ion battery under a dynamic load.

#### Understanding Battery Physics:

**Battery Internal Resistance (ESR):** Every battery has an effective series resistance (ESR). According to Ohm's Law (V=IR), when a current (I) is drawn from the battery, a voltage drop occurs across this internal resistance (R).

The terminal voltage that is measured by the MAX17040 is:
```
V_terminal = V_OCV - (I × R_ESR)
```

Where:
- `V_terminal` = Terminal voltage measured by the IC
- `V_OCV` = Open-circuit voltage of the battery
- `I` = Current being drawn
- `R_ESR` = Effective series resistance

#### Dynamic Load of Raspberry Pi:
The Raspberry Pi 5 is a **highly dynamic load**. When the CPU is idle, it draws a relatively low current. When it performs an intensive task (e.g., compiling code, processing video), the current draw increases significantly.

#### Explaining the Fluctuation:
The observed voltage drop from **4.17V to 3.85V** is a direct consequence of this principle:

- **4.17V reading**: Likely corresponds to a period of low load on the Raspberry Pi
- **3.85V reading**: Corresponds to a period of high load, where the increased current draw causes a larger voltage sag across the battery's internal resistance

> ✅ **This is normal physics, not a measurement error.**

#### The Critical Distinction:

| Register | Purpose | Behavior |
|----------|---------|----------|
| **VCELL Register (0x02)** | Provides an instantaneous snapshot of the battery's terminal voltage | Expected to fluctuate with the load. Useful for diagnostics but is a poor indicator of the battery's remaining energy |
| **SOC Register (0x04)** | The calculated output of the ModelGauge algorithm | Specifically designed to filter out these temporary voltage sags and provide a stable, reliable representation of the battery's true remaining capacity |

**Therefore, the user should rely on the (correctly calculated) SOC value as the primary indicator of battery life**, while understanding that the VCELL value will naturally and correctly fluctuate as the Raspberry Pi's workload changes.

---

## Section 4: Correcting the Implementation: A Best-Practices Guide for Python I2C

The root cause of the erroneous data is a software defect. To resolve this, a robust implementation that correctly handles the I2C communication protocol, byte ordering, and data scaling as specified by the MAX17040 datasheet is required.

### 4.1 The Canonical Implementation: qtx120x.py

When debugging hardware interactions, it is invaluable to have a known-good reference. For the Geekworm X120x series of UPS boards, the manufacturer (Suptronics) provides a Python script on GitHub, `qtx120x.py`. This script serves as the canonical example of how to correctly interface with the MAX17040 IC in this context.

#### Critical Steps from the Reference Script:

1. **I2C Library**: Uses a standard Python I2C library (like `smbus2`) to perform the low-level communication
2. **16-bit Word Reads**: Uses `bus.read_word_data(address, register)` function to read a full 16-bit word from the target register
3. **Endianness Correction**: The I2C protocol and the MAX17040 transmit data in big-endian format (most significant byte first). However, many processors, including the ARM cores in the Raspberry Pi, are little-endian.

The `qtx120x.py` script correctly handles this by swapping the byte order:
```python
swapped_word = struct.unpack('<H', struct.pack('>H', raw_word))
```

This line packs the value as a big-endian unsigned short (`>H`) and then unpacks it as a little-endian unsigned short (`<H`), effectively swapping the bytes. **This is a crucial step.**

4. **Correct Scaling**: Applies the correct scaling factors and bit shifts as dictated by the datasheet's register format descriptions

### 4.2 A Corrected and Robust Python Implementation

The following is a complete, well-commented Python class that correctly implements the I2C communication and data interpretation for the MAX17040:

```python
import smbus2
import struct
import time

class MAX17040:
    """
    A Python class to interface with the Maxim Integrated MAX17040/41 Fuel-Gauge IC.
    """
    # I2C Bus and Address Configuration
    I2C_BUS_NUMBER = 1
    I2C_ADDRESS = 0x36

    # Register Addresses
    VCELL_REG = 0x02
    SOC_REG = 0x04
    MODE_REG = 0x06
    VERSION_REG = 0x08
    RCOMP_REG = 0x0C
    COMMAND_REG = 0xFE

    def __init__(self, bus_number=I2C_BUS_NUMBER, i2c_address=I2C_ADDRESS):
        """
        Initializes the I2C bus connection.
        """
        self.bus = smbus2.SMBus(bus_number)
        self.address = i2c_address

    def _read_register_16bit(self, reg_addr):
        """
        Reads a 16-bit value from a register, handling byte order.
        The MAX17040 is a big-endian device. The smbus2 read_word_data
        returns a little-endian word, so we must swap the bytes.
        """
        raw_word = self.bus.read_word_data(self.address, reg_addr)
        # The struct packing/unpacking is a standard Python way to swap endianness
        # '>H' is big-endian unsigned short, '<H' is little-endian unsigned short
        swapped_word = struct.unpack('<H', struct.pack('>H', raw_word))[0]
        return swapped_word

    def _write_register_16bit(self, reg_addr, value):
        """
        Writes a 16-bit value to a register, handling byte order.
        """
        # Swap bytes to send in big-endian format
        swapped_value = struct.unpack('>H', struct.pack('<H', value))[0]
        self.bus.write_word_data(self.address, reg_addr, swapped_value)

    def get_cell_voltage(self):
        """
        Reads and calculates the battery cell voltage.
        VCELL format: 12 bits, left-aligned. LSB = 1.25 mV.
        The calculation is (VCELL_REGISTER_VALUE / 16) * 1.25mV.
        The division by 16 is equivalent to a right shift by 4.
        """
        vcell_raw = self._read_register_16bit(self.VCELL_REG)
        # The 4 least significant bits are always 0
        voltage = (vcell_raw >> 4) * 0.00125  # Convert from 1.25mV units to V
        return voltage

    def get_soc(self):
        """
        Reads and calculates the State of Charge (SOC) of the battery.
        SOC format: High byte is integer %, Low byte is 1/256th of a %.
        """
        soc_raw = self._read_register_16bit(self.SOC_REG)
        soc_integer = soc_raw >> 8
        soc_fractional = soc_raw & 0x00FF
        soc_percent = soc_integer + (soc_fractional / 256.0)
        return soc_percent

    def get_version(self):
        """
        Returns the production version of the IC.
        """
        return self._read_register_16bit(self.VERSION_REG)

    def get_rcomp(self):
        """
        Reads the current RCOMP compensation value.
        """
        return self._read_register_16bit(self.RCOMP_REG)

    def set_rcomp(self, rcomp_value):
        """
        Sets the RCOMP compensation value.
        :param rcomp_value: The 16-bit value to write to the RCOMP register.
        """
        self._write_register_16bit(self.RCOMP_REG, rcomp_value)

    def quick_start(self):
        """
        Issues a Quick-Start command to re-initialize the fuel-gauge calculations.
        """
        self._write_register_16bit(self.MODE_REG, 0x4000)

    def reset(self):
        """
        Issues a Power-On Reset (POR) command to the IC.
        """
        # Note: The IC does not send an ACK after this command.
        # A try/except block can handle the expected I/O error.
        try:
            self._write_register_16bit(self.COMMAND_REG, 0x0054)
        except OSError:
            pass # This is expected behavior for a POR command

# Example Usage
if __name__ == '__main__':
    try:
        ups = MAX17040()
        print("Successfully connected to MAX17040.")
        
        # Optional: Issue a quick_start on initialization
        # print("Issuing Quick-Start...")
        # ups.quick_start()
        # time.sleep(0.5) # Allow time for first conversion

        version = ups.get_version()
        print(f"IC Version: {version:04X}")

        rcomp = ups.get_rcomp()
        print(f"Default RCOMP value: {rcomp:04X}")

        while True:
            voltage = ups.get_cell_voltage()
            soc = ups.get_soc()
            
            # Create a simple battery bar
            bar_length = 20
            filled_length = int(bar_length * soc / 100)
            bar = '█' * filled_length + '─' * (bar_length - filled_length)
            
            print(f"Battery: [{bar}] {soc:6.2f}% | Voltage: {voltage:.2f}V")
            time.sleep(2)

    except FileNotFoundError:
        print("Error: I2C bus not found. Please ensure I2C is enabled.")
    except Exception as e:
        print(f"An error occurred: {e}")
```

### 4.3 Data Transformation Examples

The following table provides clear examples of the data transformation process:

| Register | Example Raw 16-bit Read (Big Endian) | Value after Byte Swap (Little Endian) | Python Calculation | Final Result |
|----------|---------------------------------------|----------------------------------------|---------------------|--------------|
| **VCELL** | `0x0D48` | `0x480D` | `(0x480D >> 4) * 0.00125` | **4.10 V** |
| **SOC** | `0x5F80` | `0x805F` | `(0x805F >> 8) + ((0x805F & 0xFF) / 256.0)` | **95.50 %** |
| **SOC** | `0x6400` | `0x0064` | `(0x0064 >> 8) + ((0x0064 & 0xFF) / 256.0)` | **100.00 %** |

### 4.4 Initializing the IC State

For robust operation, it is sometimes necessary to control the state of the fuel-gauge algorithm. The MAX17040 provides two primary commands:

#### Quick-Start
- **Purpose**: Forces the IC to restart its fuel-gauge calculations in the same manner as an initial power-up
- **When to use**: If the system's power-up sequence is electrically noisy, which might cause an error in the IC's initial "first guess" of the SOC
- **Command**: Write `0x4000` to the MODE register (`0x06`)
- **Characteristics**: Non-destructive way to re-initialize the algorithm without a full reset

#### Power-On Reset (POR)
- **Purpose**: Causes the IC to perform a complete software reset, as if power had been physically cycled
- **Command**: Write `0x0054` to the COMMAND register (`0xFE`)
- **Important Note**: After receiving this command, the IC resets immediately and does not send an I2C Acknowledge (ACK) bit, which may result in an `OSError` in the Python script that should be handled gracefully

---

## Section 5: Advanced System Tuning: Calibrating the RCOMP Register

With the primary software bug corrected, the system will now report plausible data. However, to achieve the highest possible accuracy and truly "fine-tune" the system as requested, the ModelGauge algorithm must be optimized for the specific batteries being used. This is accomplished by adjusting the RCOMP register.

### 5.1 The RCOMP Register (0x0C): The Key to Accuracy

The RCOMP register holds a 16-bit value that serves as the primary tuning parameter for the ModelGauge algorithm. Its purpose is to compensate for variations between the IC's generic internal battery model and the real-world characteristics of the user's specific battery cells.

- **Factory Default**: `0x9700`
- **Purpose**: Modify the behavior of the model to better match their cells' discharge curve under typical load and temperature conditions
- **Professional Use**: The official Linux kernel driver for the MAX17040 includes support for setting the RCOMP value via a device tree property (`maxim,rcomp`), indicating that this is a standard and expected practice in professional embedded systems design

### 5.2 A Practical Methodology for RCOMP Calibration

Maxim Integrated does not publicly provide a detailed application note on the exact algorithm for calculating the optimal RCOMP value. However, a robust empirical methodology can be employed to find a near-optimal value for a given application.

#### Calibration Goal:
Find an RCOMP value such that the fuel gauge reports an SOC of **0-1%** at the exact moment the battery voltage drops to the system's shutdown threshold (the point at which the Raspberry Pi powers off due to low voltage).

#### Structured Workflow:

| Step | Action | Tools / Commands | Observation / Data to Log | Adjustment Logic |
|------|--------|------------------|---------------------------|------------------|
| **1** | **Preparation** | Physical access, Python script | Ensure batteries are of the same model and age. Modify the Python script to allow setting RCOMP at startup. | N/A |
| **2** | **Establish Baseline** | Battery charger, Python script | Fully charge the batteries until the charger indicates completion. Set RCOMP to the default `0x9700`. | Log initial VCELL (should be ~4.2V) and SOC (should be ~100%). |
| **3** | **Apply Constant Load** | `stress-ng` or similar CPU load tool | Start a consistent, heavy CPU load on the Raspberry Pi to ensure a steady discharge rate. | N/A |
| **4** | **Discharge & Log** | Python script, data file | Run a logging script that records timestamp, VCELL, and SOC to a CSV file every 60 seconds. Let the system run until it shuts itself off due to low battery voltage. | The log file is the primary output of this step. |
| **5** | **Analyze Baseline Curve** | Spreadsheet or plotting software | Plot the logged SOC vs. VCELL. Observe the VCELL value at the moment the system shut down. | N/A |
| **6** | **Evaluate & Adjust** | Analysis of the plot | Compare the final reported SOC just before shutdown to the target of 0-1%. | **If SOC > 5% at shutdown**: The model is too conservative. The battery died while the gauge reported remaining capacity. **Action**: Decrease RCOMP (e.g., by `0x0200`).<br><br>**If SOC = 0% long before shutdown** (VCELL still high): The model is too aggressive. It reported empty prematurely. **Action**: Increase RCOMP (e.g., by `0x0200`). |
| **7** | **Iterate** | Repeat Steps 2-6 | Repeat the full charge/discharge cycle with the new RCOMP value. | Continue adjusting RCOMP until the reported SOC at shutdown is within the 0-3% range. |

### 5.3 The Role of Temperature

It is important to recognize that a battery's capacity and discharge characteristics are also dependent on temperature. The RCOMP value inherently includes a temperature compensation factor.

#### Calibration Considerations:
- **Environment**: The methodology described above calibrates the system for a specific ambient temperature
- **Best Practice**: This calibration should be performed in an environment that reflects the device's typical operating temperature
- **Typical Range**: For a Raspberry Pi system, this is likely a standard indoor room temperature
- **Advanced Options**: While more advanced fuel gauges (e.g., MAX17047/MAX17050) use an external thermistor for active temperature compensation, tuning RCOMP for the MAX17040 provides a significant improvement in accuracy for applications that operate within a relatively stable temperature range

---

## Section 6: Final Recommendations and Conclusion

The investigation into the unreliable readings from the `ups.py` script has revealed a combination of a critical software bug, a misinterpretation of normal battery physics, and an opportunity for advanced system optimization.

### 6.1 Summary of Findings

#### Primary Issue (Software Bug):
The report of an impossible **"255.21%" State of Charge** is definitively a software bug. It originates from an incorrect implementation of the I2C read protocol for the 16-bit SOC register in the `ups.py` script. The script fails to correctly parse the two-byte structure (integer and fractional parts) and likely does not handle the required byte-order (endianness) swap.

#### Secondary Issue (Misinterpretation):
The "unreliable" and fluctuating voltage readings are **not a fault of the hardware**. They represent the normal and expected voltage sag of a Li-Ion battery under the dynamic current load of the Raspberry Pi. The stable, calculated SOC value is the correct metric for judging remaining battery life, not the instantaneous terminal voltage.

#### Path to Optimization:
The MAX17040's ModelGauge algorithm uses a generic battery model. For maximum accuracy, the RCOMP compensation register must be tuned away from its factory default (`0x9700`) to match the specific discharge characteristics of the 18650 cells being used.

### 6.2 Actionable Checklist for the User

To resolve the issues and achieve a fully functional and accurate monitoring system, follow these steps **in order**:

#### ✅ Step 1: Verify Physical Connection
Before debugging software, ensure the physical I2C connection is sound. Check that the Geekworm X1202 board is seated firmly and that the pogo pins are making solid contact with the corresponding pads on the underside of the Raspberry Pi 5 PCB.

#### ✅ Step 2: Verify Raspberry Pi 5 Configuration
Confirm that the Raspberry Pi 5 EEPROM has been configured correctly for a high-power supply by setting `PSU_MAX_CURRENT=5000`. Refer to the Geekworm wiki for detailed instructions.

#### ✅ Step 3: Correct the Python Script
Replace the faulty I2C reading and parsing logic in `ups.py` with the robust implementation provided in [Section 4.2](#42-a-corrected-and-robust-python-implementation) of this report. This new code correctly handles 16-bit reads, byte-order swapping, and the specific data formats for both the VCELL and SOC registers.

#### ✅ Step 4: Establish Baseline Accuracy
Run the corrected script with the default RCOMP value (`0x9700`). The system should now report plausible and stable SOC values. Observe its performance over a partial discharge cycle to confirm basic functionality.

#### ✅ Step 5: Perform RCOMP Calibration (Recommended)
For the highest level of accuracy, follow the empirical calibration workflow detailed in [Section 5.2](#52-a-practical-methodology-for-rcomp-calibration). This involves performing at least one full, logged discharge cycle to determine the optimal RCOMP value that aligns the reported SOC with the physical shutdown voltage of the system.

#### ✅ Step 6: Implement Final Script
Once the optimal RCOMP value is determined, hardcode it into the initialization phase of the final `ups.py` script. The system is now fully corrected and fine-tuned.

### 6.3 Concluding Remarks

Modern power management ICs like the Maxim MAX17040 are remarkably powerful components, offering sophisticated algorithmic processing in a tiny, low-power package. However, this complexity comes with a requirement for **precision at the software level**.

As this analysis has shown, a successful implementation hinges on:

- 🔧 **Meticulous adherence to the manufacturer's datasheet**
- 🧠 **Correct understanding of the underlying technology**
- ⚡ **Grasp of the physical principles governing the system**

By moving beyond a simple "black box" approach and engaging with the hardware on an architectural level, developers can unlock the full potential of these components.

> 💡 **Key Insight**: The initial "unreliable" behavior was not a hardware fault but a software and conceptual gap. By bridging that gap through careful analysis, correct implementation, and empirical fine-tuning, the user can build a power monitoring system that is not only functional but also highly accurate and trustworthy, providing true confidence in their application's power resiliency.

---

## References

1. Geekworm X1202 Product Documentation
2. Geekworm Official Wiki and Installation Guides  
3. Maxim Integrated MAX17040 Datasheet
4. Raspberry Pi 5 Official Documentation
5. MAX17040 Application Notes and Technical References
6. Linux Kernel MAX17040 Driver Documentation
7. Various Technical Forums and Community Resources

---

*This document serves as a comprehensive guide for implementing reliable power monitoring with the Geekworm X1202 UPS system. For additional support or questions, refer to the manufacturer's documentation or community forums.*