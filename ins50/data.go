package ins50

type DeviceData struct {
	Data []int
}

func valid(dat []int) bool {
	return !(len(dat) < 9)
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
	return d.Data[1]
}

func (d *DeviceData) Inputs3() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[2]
}

func (d *DeviceData) Outputs1() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[3]
}

func (d *DeviceData) Outputs2() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[4]
}

func (d *DeviceData) Outputs3() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[5]
}

func (d *DeviceData) Locks1() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[6]
}

func (d *DeviceData) Locks2() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[7]
}

func (d *DeviceData) Locks3() int {
	if !valid(d.Data) {
		return -1
	}
	return d.Data[8]
}
