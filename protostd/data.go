package protostd

type DeviceData struct {
	Data []int
	Raw  []byte
}

func valid(dat []int) bool {
	return (len(dat) >= 15)
}

func (d *DeviceData) RawResponse() []byte {
	return d.Raw
}

func (d *DeviceData) Inputs1() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[0]
}

func (d *DeviceData) Inputs2() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[2]
}

func (d *DeviceData) Inputs3() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[5]
}

func (d *DeviceData) Outputs1() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[1]
}

func (d *DeviceData) Outputs2() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[3]
}

func (d *DeviceData) Outputs3() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[6]
}

func (d *DeviceData) Locks1() int {
	if !valid(d.Data) {
		return -1
	}
	if len(d.Data) > 15 {
		if d.Data[12] == 0x00 && d.Data[13] == 0x00 {
			return d.Data[14]
		}
		return d.Data[13]
	} else if len(d.Data) > 14 {
		return d.Data[10]
	}
	return 0
}

func (d *DeviceData) Locks2() int {
	if !valid(d.Data) {
		return -1
	}
	if len(d.Data) > 15 {
		if d.Data[11] == 0x00 && d.Data[12] == 0x00 && d.Data[13] == 0x00 {
			return 0x00
		}
		return d.Data[14]
	} else if len(d.Data) > 14 {
		return d.Data[13]
	}
	return 0
}

func (d *DeviceData) Locks3() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[9]
}
