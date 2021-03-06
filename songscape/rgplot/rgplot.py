#!../../env/bin/python

from scipy.io import wavfile
import numpy as np
from subprocess import call
import Image, ImageDraw, ImageChops, ImageEnhance
from os import path

import os, math, sys

FPS = 25.0


IMAGE_SIZE = (640, 480)
nFFT = IMAGE_SIZE[0]
SAMPLE_SIZE = 2
CHANNELS = 2
RATE = 44100
MAX_Y = 2.0**(SAMPLE_SIZE * 8 - 1)
BACKGROUND = (0,0,0,255)
LINE = (255, 255, 255)
LINE_START = (255, 0, 0)
LINE_END = (0, 255, 255)

LINE_HEIGHT = 180

def interpolate(i, start, end):
    return (
        int(start[0] + (end[0]-start[0]) * i),
        int(start[1] + (end[1]-start[1]) * i),
        int(start[2] + (end[2]-start[2]) * i)
    )
def hsv2rgb(h, s, v):
    h = float(h)
    s = float(s)
    v = float(v)
    h60 = h / 60.0
    h60f = math.floor(h60)
    hi = int(h60f) % 6
    f = h60 - h60f
    p = v * (1 - s)
    q = v * (1 - f * s)
    t = v * (1 - (1 - f) * s)
    r, g, b = 0, 0, 0
    if hi == 0: r, g, b = v, t, p
    elif hi == 1: r, g, b = q, v, p
    elif hi == 2: r, g, b = p, v, t
    elif hi == 3: r, g, b = p, q, v
    elif hi == 4: r, g, b = t, p, v
    elif hi == 5: r, g, b = v, p, q
    r, g, b = int(r * 255), int(g * 255), int(b * 255)
    return r, g, b

def build_fft(samples):
    samples_old = samples
    samples = samples/MAX_Y
    n_samples = len(samples)
    N = int(RATE/FPS)
    frames = int(n_samples/N)
    for frame in xrange(0, frames):
        if frame % 10 == 0:
            print('%s/%s: %s%%' % (frame, frames, frame*100/frames))
        if frame > 10:
            break
        yield np.fft.fft(samples[frame*N:(frame+1)*N], nFFT)

def main():
    arg_len = len(sys.argv)
    if arg_len < 2:
        print('Usage: spectrum.py <input file> <(optional) output file>')
    if arg_len >= 2:
        input_filename = sys.argv[1]
        output_filename = input_filename + '.mp4'
    if arg_len == 3:
        output_filename = sys.argv[2]

    call(['ffmpeg',
        '-i', input_filename,
        input_filename + '.wav'
        ])

    rate, data = wavfile.read(input_filename + '.wav')
    data = data.transpose()
    L = data[0]
    R = data[1]

    im = Image.new("RGBA", IMAGE_SIZE, BACKGROUND)
    draw = ImageDraw.Draw(im)

    line = Image.new("RGBA", (1, LINE_HEIGHT))
    line_draw = ImageDraw.Draw(line)
    for y in xrange(LINE_HEIGHT):
        fill = hsv2rgb(float(y)/LINE_HEIGHT*120,1,1)
        # fill = interpolate(float(y)/LINE_HEIGHT, LINE_START, LINE_END)
        line_draw.point((0,y), fill = fill)

    try:
        for i, fft in enumerate(build_fft(L)):
            x_offset = 0
            y_offset = 5
            width, height = im.size
            im = ImageChops.offset(im, x_offset, -y_offset)

            enhancer = ImageEnhance.Brightness(im)
            im = enhancer.enhance(.98)

            #pix = im.load()
            draw = ImageDraw.Draw(im)
            draw.rectangle((0, 0, x_offset, height), BACKGROUND)
            draw.rectangle((0, height + y_offset, width, height), BACKGROUND)

            prev = None
            for x, y in enumerate(fft):
                # y = math.sqrt(abs(y.real) * 1e2) * 10
                # y = abs(y.real) * 10
                y = math.log(abs(y.real) + 1) * 35
                # print(y)
                y = min(LINE_HEIGHT, y) 
                #fill=hsv2rgb((i*3) % 360, 1, 1)
                #fill = (255-int(y*1.3),)*3
                #print(fill)
                #draw.line((x_, height, x_, height-y), fill=fill) #interpolate(float(x)/nFFT, LINE_START, LINE_END))
                from_box = (0, LINE_HEIGHT - int(y), 1, LINE_HEIGHT)
                region = line.crop(from_box)

                x = (x + nFFT/2) % nFFT
                x += (im.size[0] - nFFT) / 2
                # y_bot = height - (nFFT/2 - x/2) # Slanty
                y_bot = height

                to_box = (x, y_bot - int(y), x+1, y_bot)

                #to_box = (0, height - int(y) - (nFFT - x), 1, height - (nFFT - x))
                im.paste(region, to_box)

                if prev:
                    draw.line((x-1, y_bot - prev, x, y_bot - y), fill=(0,0,0,0))
                prev = y
            im.save('tmp/%07d.png' % i)
    except KeyboardInterrupt:
        pass
    call(["ffmpeg",
            "-y",
            "-i", input_filename,
            "-shortest",
            "-r", str(FPS),
            "-i", 'tmp/%07d.png',
            "-pix_fmt", "yuv420p",
            "-r", str(FPS),
            output_filename])

    if  path.exists(output_filename):
        os.system('rm tmp/*')
    os.unlink(input_filename + '.wav')

if __name__ == "__main__":
    main()
#Y_R = np.fft.fft(y_R, nFFT)

