package game

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

// Sounds is every sound Reflex plays, keyed by its path in client/: the
// arcade kit, and its own (the bell through WAIT..., the word, the bang,
// losing). release.go packs them.
func Sounds() map[string][]byte {
	out := arcade.SoundKit()
	out[sndTick] = wav(.05, func(t float64) float64 { // a clock's tick
		return .4*math.Sin(2*math.Pi*1500*t)*math.Exp(-t*90) + .2*math.Sin(2*math.Pi*3100*t)*math.Exp(-t*160)
	})
	out[sndDraw] = wav(.32, func(t float64) float64 { // the word: two high notes
		f := 1568.0
		if t > .12 {
			f = 2093
		}
		return .3 * square(f, t) * math.Exp(-math.Mod(t, .12)*6)
	})
	out[sndBang] = wav(.45, func(t float64) float64 { // a shot: noise and a falling thump
		return .5*noise(t)*math.Exp(-t*14) + .4*math.Sin(2*math.Pi*(180-200*t)*t)*math.Exp(-t*9)
	})
	out[sndLose] = wav(.6, func(t float64) float64 { // down three notes
		notes := []float64{392, 330, 262}
		i := min(int(t/.18), len(notes)-1)
		return .3 * square(notes[i], t) * math.Exp(-(t-float64(i)*.18)*7)
	})
	return out
}

func square(f, t float64) float64 { return math.Tanh(4 * math.Sin(2*math.Pi*f*t)) }

// noise is repeatable white noise.
func noise(t float64) float64 {
	x := math.Sin(t*12345.678) * 43758.5453
	return 2*(x-math.Floor(x)) - 1
}

const rate = 22050

// wav renders f over dur seconds as a 16-bit mono WAV file.
func wav(dur float64, f func(t float64) float64) []byte {
	n := int(dur * rate)
	data := make([]byte, 2*n)
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		fade := math.Min(1, (dur-t)/.01)
		v := max(-1, min(1, f(t)*fade))
		binary.LittleEndian.PutUint16(data[2*i:], uint16(int16(v*32000)))
	}
	var b bytes.Buffer
	b.WriteString("RIFF")
	_ = binary.Write(&b, binary.LittleEndian, uint32(36+len(data)))
	b.WriteString("WAVEfmt ")
	for _, v := range []any{uint32(16), uint16(1), uint16(1), uint32(rate), uint32(rate * 2), uint16(2), uint16(16)} {
		_ = binary.Write(&b, binary.LittleEndian, v)
	}
	b.WriteString("data")
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(data)))
	b.Write(data)
	return b.Bytes()
}
