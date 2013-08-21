package main

import (
	//"github.com/sdobz/go-dsp/fft"
	"bytes"
	bin "encoding/binary"
	"fmt"
	"github.com/sdobz/go-wav"
	"github.com/runningwild/go-fftw"
	"io"
	"os"
	"os/exec"
)

type Sample struct {
	L, R int16
}

func fill_color(buf []byte, col []uint8) {
	for i := 0; i < len(buf); i += 4 {
		copy(buf[i:i+4], col)
	}
}

func main() {
	FPS := 25
	WIDTH, HEIGHT := uint32(640), uint32(480)
	NFFT := WIDTH

	// b g r a
	BACKGROUND := []uint8{0, 0, 0, 255}
	WHITE := []uint8{255, 255, 255, 255}

	height_f := float64(HEIGHT)
	cmd := exec.Command("ffmpeg",
		"-y",
		"-f", "rawvideo",
		"-pixel_format", "rgb32",
		"-video_size", fmt.Sprintf("%dx%d", WIDTH, HEIGHT),
		"-framerate", fmt.Sprintf("%d", FPS),
		"-i", "-",
		"-i", "starfish.wav",
		"-pix_fmt", "yuv420p",
		"-preset", "ultrafast",
		"-shortest",
		"output.mp4")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println(err)
	}

	wav_data := wav.ReadWavData("starfish.wav") // For read access.

	SAMPLES := uint32(len(wav_data.Data)) / (uint32(wav_data.BitsPerSample) / 8) / uint32(wav_data.NumChannels)

	// Samples per frame to step
	SPF := wav_data.SampleRate / uint32(FPS)

	var N uint32
	// Samples per frame to take
	if NFFT > SPF {
		N = SPF
	} else {
		N = NFFT
	}

	max_y := float64((uint32(1)<<wav_data.BitsPerSample) - 1)

	var wav_buf *bytes.Buffer

	buf := make([]byte, WIDTH*HEIGHT*4)
	fft_data := fftw.Alloc1d(int(NFFT))
	fft_f := fftw.PlanDft1d(fft_data, fft_data, fftw.Forward, fftw.Estimate)

	var wav_offset, buf_offset uint32
	var y uint32
	var tmp_y float64
	var sample Sample
	for frame := uint32(0); frame < SAMPLES/SPF; frame++ {
		fmt.Printf("Frame: %d/%d: %d%%\n", frame, SAMPLES/SPF, frame*100/(SAMPLES/SPF))

		fill_color(buf, BACKGROUND)

		wav_offset = frame * SPF * 4

		wav_buf = bytes.NewBuffer(wav_data.Data[wav_offset : wav_offset+(SPF*4)])

		for x := uint32(0); x < N; x++ {
			bin.Read(wav_buf, bin.LittleEndian, &sample)
			fft_data[x] = complex(float64(sample.L)/max_y,float64(0.0))
			//y = uint32((int32(sample.L) * int32(HEIGHT) / int32(max_y)) + int32(HEIGHT)/2)
			//buf_offset = y*WIDTH*4 + x*4
			//copy(buf[buf_offset:buf_offset+4], WHITE)
		}
		//fmt.Printf("\n")

		fft_f.Execute()
		for x := uint32(0); x < N; x++ {
			tmp_y = height_f - real(fft_data[x]) * 80.0
			//fmt.Println(tmp_y)

			if tmp_y < 0 {
				continue
			} 
			if tmp_y > height_f - 1 {
				tmp_y = height_f - 1
			}
			y = uint32(tmp_y)
			buf_offset = y*WIDTH*4 + x*4
			copy(buf[buf_offset:buf_offset+4], WHITE)
		}

		if _, err := stdin.Write(buf); err != nil {
			panic(err)
		}
	}
	stdin.Close()
	cmd.Wait()

	io.Copy(os.Stdout, stdout)
	io.Copy(os.Stderr, stderr)
	fmt.Printf("Command finished with error: %v\n", err)
}