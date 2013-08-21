// Copyright 2012 The go-gl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This test opens a window with FSAA sampling enabled, then verifies that we
// indeed got a window with > 0 sampling enabled. Creation of the window is
// necessary to make the test work.
package main

import (
	"fmt"
	"log"

	"github.com/go-gl/gl"
	"github.com/go-gl/glh"
	"github.com/go-gl/glfw"
	"github.com/go-gl/glu"
	"github.com/go-gl/gltext"

	"github.com/sdobz/go-wav"
	"github.com/runningwild/go-fftw"
	"bytes"
	bin "encoding/binary"
	"github.com/andrebq/gas"
	"os"

	"github.com/sdobz/Go-SDL/mixer"
)

const GL_MULTISAMPLE_ARB = 0x809D

const (
	WIDTH = 640
	HEIGHT = 480
	NFFT = 640
)

type Sample struct {
	L, R int16
}
var font *gltext.Font

// loadFont loads the specified font at the given scale.
func loadFont(file string, scale int32) (*gltext.Font, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer fd.Close()

	return gltext.LoadTruetype(fd, scale, 32, 127, gltext.LeftToRight)
}

// drawString draws the same string for each loaded font.
func draw_string(x, y float32, str string) error {
	// Render the string.
	gl.Color4f(1, 1, 1, 1)
	err := font.Printf(x, y, str)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	err := initGL()
	if err != nil {
		log.Fatalf("InitGL: %v", err)
		return
	}
	defer glfw.Terminate()

	file, err := gas.Abs("code.google.com/p/freetype-go/luxi-fonts/luxisr.ttf")
	if err != nil {
		log.Printf("Find font file: %v", err)
		return
	}

	// Load the same font at different scale factors and directions.
	font, err = loadFont(file, int32(12))
	if err != nil {
		log.Printf("LoadFont: %v", err)
		return
	}
	defer font.Release()


	// Open window with FSAA samples (if possible).
	// glfw.OpenWindowHint(glfw.FsaaSamples, 4)

	wav_data := wav.ReadWavData("starfish.wav") // For read access.
	bpsa := uint32(wav_data.NumChannels * (wav_data.BitsPerSample/8))
	bpse := float32(wav_data.SampleRate * bpsa)
	max_y := float64(int(1) << wav_data.BitsPerSample)

	fft_data := fftw.Alloc1d(int(NFFT))
	fft_f := fftw.PlanDft1d(fft_data, fft_data, fftw.Forward, fftw.Estimate)
	
	// GO-SDL
	mixer.OpenAudio(int(wav_data.SampleRate), mixer.AUDIO_S16LSB, int(wav_data.NumChannels), 2048)
	music := mixer.LoadMUS("starfish.wav")
	defer music.Free()
	//mixer.SetPostMix(&callback)
	//callback := func (data []byte) {
	//	fmt.Printf("Callback got: %d bytes\n", len(data))
	//}
	music.PlayMusic(1)
	
	var wav_buf *bytes.Buffer
	var wav_offset uint32
	var sample Sample
	var y float32
	var fps int
	last_time := float32(glfw.Time()) + 0.1

	// Width of a bar, in order to fit NFFT bars in one 2-wide screen
	bar_w := 1.0/float32(NFFT)

	for glfw.WindowParam(glfw.Opened) == 1 {
		time := float32(glfw.Time())
		//time -= 2
		if time < 0 {
			continue
		}
		fps = (fps * 90 + int(10.0/(time - last_time))) / 100
		last_time = time

		// This will be how far into the file to look
		wav_offset = uint32(time * bpse)
		// Get a buffer of the samples for this frame
		wav_buf = bytes.NewBuffer(wav_data.Data[wav_offset : wav_offset + WIDTH*bpsa])
		for i := 0; i < NFFT; i++ {
			bin.Read(wav_buf, bin.LittleEndian, &sample)
			fft_data[i] = complex(float64(sample.L)/max_y,float64(0.0))
		}
		fft_f.Execute()

		gl.Clear(gl.COLOR_BUFFER_BIT)

		err = draw_string(0,0,fmt.Sprintf("%d",fps))
		if err != nil {
			log.Printf("Printf: %v", err)
			return
		}

		gl.Color3f(1, 1, 1)
		for i := 0; i < NFFT; i++ {
			y = float32(real(fft_data[(i + NFFT/2) % NFFT])) / 10.0
			if y > 2 {
				y = 2
			}
			gl.LoadIdentity()
			gl.Translatef(float32(i)/(NFFT/2) - 1, 0, 0)
			gl.Rectf(-bar_w, -1 + y, bar_w, -1)
		}
		
		// gl.Rotatef(time, 0, 0, 1)

		// gl.Enable(GL_MULTISAMPLE_ARB)
		// gl.Disable(GL_MULTISAMPLE_ARB)
		glfw.SwapBuffers()
	}
}

// initGL initializes GLFW and OpenGL.
func initGL() error {
	err := glfw.Init()
	if err != nil {
		return err
	}

	if err = glfw.OpenWindow(WIDTH, HEIGHT, 8, 8, 8, 8, 0, 0, glfw.Windowed); err != nil {
		log.Fatalf("%v\n", err)
		return err
	}

	glfw.SetWindowSizeCallback(onResize)
	glfw.SetKeyCallback(onKey)
	glfw.SetWindowTitle("RGPlot")
	glfw.SetSwapInterval(1)

	gl.MatrixMode(gl.PROJECTION)
	glu.Perspective(0, 1, 0, 1)
	// Above line causes error, clear it below
	glh.CheckGLError();

	errno := gl.Init()
	if errno != gl.NO_ERROR {
		str, err := glu.ErrorString(errno)
		if err != nil {
			return fmt.Errorf("Unknown openGL error: %d", errno)
		}
		return fmt.Errorf(str)
	}

	gl.Disable(gl.DEPTH_TEST)

	gl.Disable(gl.LIGHTING)
	gl.ClearColor(0.2, 0.2, 0.23, 0.0)
	return nil
}

// onKey handles key events.
func onKey(key, state int) {
	if key == glfw.KeyEsc {
		glfw.CloseWindow()
	}
}

// onResize handles window resize events.
func onResize(w, h int) {
	if w < 1 {
		w = 1
	}

	if h < 1 {
		h = 1
	}

	gl.Viewport(0, 0, w, h)
	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Ortho(0, float64(w), float64(h), 0, 0, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.LoadIdentity()
}