package main

func generateBeepPayload() []byte {
	data := make([]byte, 160)

	for i := 0; i < len(data); i++ {
		if i%20 < 10 {
			data[i] = 0xFF
		} else {
			data[i] = 0x7F
		}
	}

	return data
}
